package hive

import (
    "encoding/gob"
    "fmt"
    "net/rpc"
    "os/exec"

    "github.com/hashicorp/go-hclog"
    "github.com/hashicorp/go-plugin"
)

type Handler interface {
    On(c *Context)
}

type Context struct {
    request  *CallRequest
    response CallResponse
}
type CallRequest struct {
    Name string
    Data []byte
}
type CallResponse struct {
    Code int
    Data []byte
    Err  string
}
type Bee struct {
    name   string
    client *plugin.Client
}
type Option struct {
    Name       string
    PluginPath string
    Magic      Magic
}
type Magic struct {
    Key   string
    Value string
}

var defaultMagic = &Magic{
    Key:   "HivePluginMagicKey",
    Value: "HivePluginMagicValue",
}

func init() {
    gob.Register(CallRequest{})
    gob.Register(CallResponse{})
}

func (c *Context) Name() string {
    return c.request.Name
}
func (c *Context) Data() []byte {
    return c.request.Data
}
func (c *Context) Reply(code int, data []byte) {
    c.response.Code = code
    c.response.Data = data
}

func RegisterPlugin(ptr Handler, name string, m *Magic, logger hclog.Logger) error {
    if m == nil {
        m = defaultMagic
    }
    var handshakeConfig = plugin.HandshakeConfig{
        ProtocolVersion:  1,
        MagicCookieKey:   m.Key,
        MagicCookieValue: m.Value,
    }
    // pluginMap is the map of plugins we can dispense.
    var pluginMap = map[string]plugin.Plugin{
        name: &callPlugin{Impl: ptr},
    }
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: handshakeConfig,
        Plugins:         pluginMap,
        Logger:          logger,
    })
    return nil
}

func StartPlugin(name, pluginPath string, m *Magic) *Bee {
    if m == nil {
        m = defaultMagic
    }
    var handshakeConfig = plugin.HandshakeConfig{
        ProtocolVersion:  1,
        MagicCookieKey:   m.Key,
        MagicCookieValue: m.Value,
    }
    var pluginMap = map[string]plugin.Plugin{
        name: &callPlugin{},
    }
    client := plugin.NewClient(&plugin.ClientConfig{
        HandshakeConfig: handshakeConfig,
        Plugins:         pluginMap,
        Cmd:             exec.Command(pluginPath),
    })

    b := &Bee{
        name:   name,
        client: client,
    }
    return b
}
func (c *Bee) Invoke(name string, data []byte) (int, []byte, error) {
    client := c.client
    rpcClient, err := client.Client()
    if err != nil {
        return 0, nil, err
    }
    raw, err := rpcClient.Dispense(c.name)
    if err != nil {
        return 0, nil, err
    }
    if r, ok := raw.(*callerRPC); ok && r != nil {
        resp, err := r.Invoke(name, data)
        if err != nil {
            return 0, nil, err
        }
        if resp.Err != "" {
            return resp.Code, resp.Data, fmt.Errorf("%v", resp.Err)
        }
        return resp.Code, resp.Data, nil
    }

    return 0, nil, fmt.Errorf("type error%+v", raw)
}
func (c *Bee) Close() {
    c.client.Kill()
}

type callerRPC struct{ client *rpc.Client }

func (c *callerRPC) Invoke(name string, data []byte) (*CallResponse, error) {
    var req = new(CallRequest)
    var reply CallResponse
    req.Name = name
    req.Data = data
    err := c.client.Call("Plugin.Invoke", req, &reply)
    if err != nil {
        return nil, err
    }
    return &reply, nil
}

type callerServer struct {
    Impl Handler
}

func (s *callerServer) Invoke(args *CallRequest, resp *CallResponse) (err error) {
    defer func() {
        if e := recover(); e != nil {
            resp.Err = fmt.Sprintf("%v", e)
        }
    }()
    ctx := &Context{
        request: args,
    }
    s.Impl.On(ctx)
    *resp = ctx.response
    return
}

type callPlugin struct {
    Impl Handler
}

func (p *callPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
    return &callerServer{Impl: p.Impl}, nil
}
func (p *callPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
    return &callerRPC{client: c}, nil
}
