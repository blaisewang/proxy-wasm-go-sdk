package main

import (
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

func main() {
	proxywasm.SetVMContext(NewVMContext())
}

type vmContext struct {
	types.DefaultVMContext
}

func NewVMContext() types.VMContext {
	return &vmContext{}
}

func (ctx *vmContext) OnVMStart(vmConfigurationSize int) types.OnVMStartStatus {
	return types.OnVMStartStatusOK
}

type pluginContext struct {
	types.DefaultPluginContext
}

func (ctx *vmContext) NewPluginContext(_ uint32) types.PluginContext {
	return &pluginContext{}
}

func (ctx *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	return types.OnPluginStartStatusOK
}

type httpContext struct {
	types.DefaultHttpContext
	contextID       uint32
	headers         [][2]string
	processed       bool
	requestBodySize int
}

func (ctx *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpContext{
		contextID: contextID,
	}
}

func (ctx *httpContext) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	if ctx.processed {
		return types.ActionContinue
	}

	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("ctx_id: %d failed to get response headers: %v", ctx.contextID, err)
		ctx.processed = true
		return types.ActionContinue
	}

	if endOfStream {
		ctx.processed = true
		return ctx.dispatchDetection(headers, nil)
	}

	ctx.headers = headers

	return types.ActionContinue
}

func (ctx *httpContext) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	if ctx.processed {
		return types.ActionContinue
	}

	ctx.requestBodySize += bodySize
	if !endOfStream {
		return types.ActionPause
	}

	body, err := proxywasm.GetHttpRequestBody(0, ctx.requestBodySize)
	if err != nil {
		proxywasm.LogCriticalf("ctx_id: %d failed to get request body: %v", ctx.contextID, err)
		ctx.processed = true
		return types.ActionContinue
	}

	return ctx.dispatchDetection(ctx.headers, body)
}

func (ctx *httpContext) dispatchDetection(headers [][2]string, body []byte) types.Action {
	if calloutID, err := proxywasm.DispatchHttpCall(
		"detector",
		headers,
		body,
		nil,
		1000,
		ctx.httpCallResponseCallback,
	); err != nil {
		proxywasm.LogErrorf("ctx_id: %d failed to dispatch http call %d: %v", ctx.contextID, calloutID, err)
		return types.ActionContinue
	}

	return types.ActionPause
}

func (ctx *httpContext) httpCallResponseCallback(numHeaders, bodySize, numTrailers int) {
	resHeaders, err := proxywasm.GetHttpCallResponseHeaders()
	if err != nil {
		proxywasm.LogCriticalf("ctx_id: %d failed to get response headers, resuming request: %v", ctx.contextID, err)
		_ = proxywasm.ResumeHttpRequest()
		return
	}

	// save headers to map
	resMap := make(map[string]string)
	for _, v := range resHeaders {
		resMap[v[0]] = v[1]
	}

	result, ok := resMap["x-filter-result"]
	if !ok || result != "?" {
		proxywasm.LogDebugf("ctx_id: %d result is empty or not blocked, resuming request: %s", ctx.contextID, result)
		_ = proxywasm.ResumeHttpRequest()
		return
	}

	body := "access forbidden"
	proxywasm.LogInfo(body)
	if err := proxywasm.SendHttpResponse(403, [][2]string{
		{"powered-by", "proxy-wasm-go-sdk!!"},
	}, []byte(body), -1); err != nil {
		proxywasm.LogErrorf("failed to send local response: %v", err)
		_ = proxywasm.ResumeHttpRequest()
	}

}
