package plugins

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cjoudrey/gluahttp"
	"github.com/layeh/gopher-json"
	"github.com/topfreegames/mqttbot/logger"
	"github.com/topfreegames/mqttbot/plugins/modules"
	"github.com/yuin/gopher-lua"
)

type Plugins struct {
	PluginMappings []map[string]string
}

func GetPlugins() *Plugins {
	plugins := &Plugins{
		PluginMappings: []map[string]string{},
	}
	return plugins
}

func (p *Plugins) SetupPlugins() {
	p.preloadModules()
}

func (p *Plugins) preloadModules() {
	L := lua.NewState()
	defer L.Close()
	p.loadModules(L)
	if err := L.DoFile("plugins/load_modules.lua"); err != nil {
		logger.Logger.Fatal("Error loading lua go modules, err:", err)
	}
	logger.Logger.Info("Successfully loaded lua go modules")
}

func (p *Plugins) loadModules(L *lua.LState) {
	L.PreloadModule("persistence_module", modules.PersistenceModuleLoader)
	L.PreloadModule("mqttclient_module", modules.MqttClientModuleLoader)
	L.PreloadModule("redis_module", modules.RedisModuleLoader)
	L.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	L.PreloadModule("json", json.Loader)
}

func (p *Plugins) ExecutePlugin(payload, topic, plugin string) (err error, success int) {
	L := lua.NewState()
	p.loadModules(L)
	L.DoFile(fmt.Sprintf("./plugins/%s.lua", plugin))
	defer L.Close()
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("run_plugin"),
		NRet:    2,
		Protect: true,
	}, lua.LString(topic), lua.LString(payload)); err != nil {
		logger.Logger.Error(err)
		return err, 1
	}
	ret := L.Get(-1)
	retErr := L.Get(-2)
	L.Pop(2)
	if retErr != nil && retErr != lua.LNil {
		logger.Logger.Error(retErr.String())
		return errors.New(retErr.String()), 1
	}
	return nil, int(ret.(lua.LNumber))
}
