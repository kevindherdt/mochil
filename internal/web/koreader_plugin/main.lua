-- Mochil Sync loader. It stays tiny and rarely changes: the plugin itself lives in mochil_core.lua, which
-- can be replaced from inside KOReader. If a new core fails to load, the previous one is used instead.
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local InfoMessage = require("ui/widget/infomessage")
local UIManager = require("ui/uimanager")
local logger = require("logger")
local _ = require("gettext")

local LOADER_API = 1
local source = debug.getinfo(1, "S").source or ""
local DIR = source:match("^@(.*/)[^/]*$") or "./"
local CORE, BACKUP = "mochil_core.lua", "mochil_core.lua.bak"

local function load_core(name)
    local chunk, err = loadfile(DIR .. name)
    if not chunk then return nil, err end
    local ok, core = pcall(chunk, {api = LOADER_API, dir = DIR, core_file = CORE, backup_file = BACKUP})
    if ok and type(core) == "table" then return core end
    return nil, core
end

local function copy_file(from, to)
    local input = io.open(from, "rb"); if not input then return false end
    local data = input:read("*a"); input:close()
    local output = io.open(to, "wb"); if not output then return false end
    output:write(data); output:close()
    return true
end

local core, err = load_core(CORE)
if not core then
    logger.err("Mochil: the plugin could not start, trying the previous version:", tostring(err))
    core = load_core(BACKUP)
    -- Put the working copy back so the next start does not need the fallback.
    if core then
        pcall(copy_file, DIR .. BACKUP, DIR .. CORE)
        -- Remember why, so the plugin can say so instead of quietly staying on the old version.
        core.fallback_error = tostring(err):sub(1, 300)
    end
end
if core then return core end

local Broken = WidgetContainer:extend{name = "mochil", is_doc_only = false}
function Broken:init() pcall(function() self.ui.menu:registerToMainMenu(self) end) end
function Broken:addToMainMenu(menu_items)
    menu_items.mochil = {text = _("Mochil Sync (reinstall needed)"), sorting_hint = "tools", callback = function()
        UIManager:show(InfoMessage:new{text = _("Mochil Sync could not start. Download the plugin again from Mochil Settings and replace this folder."), timeout = 8})
    end}
end
return Broken
