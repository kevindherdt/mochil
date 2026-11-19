-- A stand-in for the parts of KOReader the plugin touches, so its logic can run under plain Lua.
local H = {closed = {}, logs = {}, shown = {}, scheduled = {}, requests = {}, timeouts = {}, events = {}}

-- Just enough JSON for the plugin's requests and the test's replies.
local function encode(value)
    local kind = type(value)
    if kind == "nil" then return "null" end
    if kind == "boolean" or kind == "number" then return tostring(value) end
    if kind == "string" then return '"' .. value:gsub('[%c"\\]', function(c) return string.format("\\u%04x", c:byte()) end) .. '"' end
    if kind ~= "table" then error("cannot encode " .. kind) end
    if #value > 0 or next(value) == nil then
        local out = {}
        for _, item in ipairs(value) do out[#out + 1] = encode(item) end
        return "[" .. table.concat(out, ",") .. "]"
    end
    local out = {}
    for key, item in pairs(value) do out[#out + 1] = encode(tostring(key)) .. ":" .. encode(item) end
    return "{" .. table.concat(out, ",") .. "}"
end
local function decode(text)
    local position = 1
    local function skip() position = text:find("%S", position) or #text + 1 end
    local parse
    function parse()
        skip()
        local char = text:sub(position, position)
        if char == "{" then
            local result = {}; position = position + 1; skip()
            if text:sub(position, position) == "}" then position = position + 1; return result end
            repeat
                local key = parse(); skip(); assert(text:sub(position, position) == ":"); position = position + 1
                result[key] = parse(); skip()
                local sep = text:sub(position, position); position = position + 1
            until sep == "}"
            return result
        elseif char == "[" then
            local result = {}; position = position + 1; skip()
            if text:sub(position, position) == "]" then position = position + 1; return result end
            repeat
                result[#result + 1] = parse(); skip()
                local sep = text:sub(position, position); position = position + 1
            until sep == "]"
            return result
        elseif char == '"' then
            local finish = position + 1
            while text:sub(finish, finish) ~= '"' do finish = finish + (text:sub(finish, finish) == "\\" and 2 or 1) end
            local raw = text:sub(position + 1, finish - 1); position = finish + 1
            return (raw:gsub("\\u(%x%x%x%x)", function(h) return utf8.char(tonumber(h, 16)) end):gsub("\\(.)", "%1"))
        elseif text:sub(position, position + 3) == "true" then position = position + 4; return true
        elseif text:sub(position, position + 4) == "false" then position = position + 5; return false
        elseif text:sub(position, position + 3) == "null" then position = position + 4; return nil
        end
        local number = text:match("^-?%d+%.?%d*[eE]?[+-]?%d*", position)
        assert(number and #number > 0, "bad JSON at " .. position)
        position = position + #number
        return tonumber(number)
    end
    local value = parse(); skip()
    assert(position > #text, "trailing JSON")
    return value
end
H.encode, H.decode = encode, decode

local function extend(base, fields)
    fields = fields or {}
    fields.__index = fields
    return setmetatable(fields, {__index = base, __call = function(class, o) return class:new(o) end})
end
local Widget = {}
function Widget:extend(fields) return extend(self, fields) end
function Widget:new(o)
    o = o or {}
    setmetatable(o, {__index = self})
    if o.init then o:init() end
    return o
end

local function preload(name, value) package.preload[name] = function() return value end end

preload("ui/widget/container/widgetcontainer", Widget)
preload("ui/widget/confirmbox", Widget:extend{})
preload("ui/widget/infomessage", Widget:extend{})
preload("ui/widget/textviewer", {new = function(_, fields) return fields end})
preload("ui/widget/buttondialog", Widget:extend{})
preload("ui/widget/inputdialog", Widget:extend{getInputText = function(self) return self.input_text end})
preload("ui/widget/pathchooser", Widget:extend{})
H.registered_actions = {}
preload("dispatcher", {registerAction = function(self, name, spec) H.registered_actions[name] = spec end})
H.notified = {}
preload("ui/widget/notification", {notify = function(self, text) H.notified[#H.notified + 1] = text end})
-- ImageViewer: KOReader's full-screen paginated image widget, used by Stream. The test harness
-- keeps it minimal (no real rendering); tests inspect the fields the plugin passed in.
preload("ui/widget/imageviewer", Widget:extend{
    switchToImageNum = function(self, num) self._shown_image_num = num end,
})
-- RenderImage: in real KOReader this decodes bytes into a bitmap. The stand-in just remembers
-- what it was given, so a test can tell a successful page fetch from the error placeholder.
preload("ui/renderimage", {
    renderImageData = function(_, data, _, _) return {_rendered_from = data} end,
    renderImageFile = function(_, path, _) return {_rendered_from_file = path} end,
})
H.opened = {}
preload("apps/reader/readerui", {showReader = function(_, path) H.opened[#H.opened + 1] = path end})
preload("device", {model = "TestReader", home_dir = "/tmp/mochil-test-home", screen = {getWidth = function() return 600 end, getHeight = function() return 800 end, scaleBySize = function(_, n) return n end}})
-- Every temp dir this suite creates is tracked and removed: each test used to leave two /tmp/lua_* dirs
-- behind for good, and where /tmp is a tmpfs those leftovers *are* host RAM — ~100k of them once pushed
-- the kernel into a global OOM. They are cleaned on each reset and on the way out of the process.
H.tempdirs, H.alltemp = {}, {}
function H.tmpdir()
    local dir = os.tmpname(); os.remove(dir); assert(os.execute("mkdir -p " .. dir))
    H.tempdirs[#H.tempdirs + 1] = dir
    H.alltemp[#H.alltemp + 1] = dir
    return dir
end
function H.cleanTemp()
    for _, dir in ipairs(H.tempdirs) do os.execute("rm -rf " .. dir) end
    H.tempdirs = {}
end
do
    local osExit = os.exit
    os.exit = function(...)
        H.cleanTemp()
        -- A test can write into a path a *later* test had already cleaned from the settings store, so the
        -- exit sweep covers every dir the process ever made, not just the last test's.
        if #H.alltemp > 0 then os.execute("rm -rf " .. table.concat(H.alltemp, " ")) end
        H.alltemp = {}
        return osExit(...)
    end
end
-- The download root lives under the device home dir, so a leftover one from an earlier run would decide
-- whether this run can cache covers (and remember a cover miss for good). Each test gets its own.
local function freshHome()
    H.home = H.tmpdir()
    local Device = package.loaded["device"]; if Device then Device.home_dir = H.home end
    return H.home
end
freshHome()
-- KOReader's full-screen list: it keeps what it was given and is "shown" through UIManager like any widget.
-- The widgets the designed page is built from: each keeps what it was given, so a test can read the page back.
for _, name in ipairs{"ui/widget/container/inputcontainer", "ui/widget/container/framecontainer", "ui/widget/verticalgroup",
        "ui/widget/verticalspan", "ui/widget/horizontalspan", "ui/widget/horizontalgroup", "ui/widget/linewidget",
        "ui/widget/textwidget", "ui/widget/titlebar", "ui/widget/buttontable", "ui/widget/progresswidget", "ui/geometry", "ui/gesturerange",
        "ui/widget/imagewidget", "ui/widget/textboxwidget"} do
    H.widgetModules = H.widgetModules or {}
    H.widgetModules[name] = function() return {new = function(_, fields) return fields or {} end} end
    if name == "ui/widget/textwidget" then
        -- Real KOReader TextWidget: always a single line (never grows taller for long text) and, when
        -- max_width is given with truncate_with_ellipsis (the real default), the *drawn* text is cut short
        -- with a trailing "…". The budget below is deliberately generous (character count, not real font
        -- metrics) so ordinary titles/authors used elsewhere in this suite are never touched; only text that
        -- would clearly overflow a card gets shortened, matching what real device truncation is for.
        H.widgetModules[name] = function()
            return {new = function(_, fields)
                fields = fields or {}
                fields._full_text = fields.text
                if type(fields.text) == "string" and fields.max_width and fields.truncate_with_ellipsis ~= false then
                    local budget = math.floor(fields.max_width / 4)
                    if budget > 0 and #fields.text > budget then
                        fields.text = fields.text:sub(1, math.max(1, budget - 1)) .. "…"
                        fields._is_truncated = true
                    end
                end
                fields.getSize = fields.getSize or function(self) return {w = math.min(#self.text * 4, self.max_width or math.huge), h = 10} end
                return fields
            end}
        end
    elseif name == "ui/widget/verticalgroup" then
        -- Like KOReader's: the layout is worked out once and remembered, so items added after that have no position.
        H.widgetModules[name] = function()
            local Group = {}
            Group.__index = Group
            function Group:new(fields) return setmetatable(fields or {}, Group) end
            function Group:getSize()
                if not self._size then
                    self._size = {w = 0, h = 0}; self._offsets = {}
                    for i, widget in ipairs(self) do
                        local size = widget.getSize and widget:getSize() or {w = 0, h = 10}
                        self._offsets[i] = {x = size.w, y = self._size.h}
                        self._size.h = self._size.h + size.h
                    end
                end
                return self._size
            end
            function Group:resetLayout() self._size = nil; self._offsets = {} end
            function Group:paintTo(bb, x, y)
                self:getSize()
                for i, widget in ipairs(self) do
                    local at = y + self._offsets[i].y -- fails for an item added after the layout was remembered
                    if widget.paintTo then widget:paintTo(bb, x, at) end
                end
            end
            return Group
        end
    elseif name:find("container") then
        H.widgetModules[name] = function()
            return {new = function(_, fields)
                fields = fields or {}
                fields.paintTo = function(self, bb, x, y) for _, child in ipairs(self) do if type(child) == "table" and child.paintTo then child:paintTo(bb, x, y) end end end
                fields.getSize = fields.getSize or function(self) return {w = 0, h = 0} end
                return fields
            end}
        end
    end
    package.preload[name] = H.widgetModules[name]
end
preload("ui/font", {getFace = function(_, name, size) return {name = name, size = size} end})
preload("ffi/blitbuffer", {COLOR_WHITE = 1, COLOR_BLACK = 0, COLOR_GRAY = 2, COLOR_DARK_GRAY = 3, COLOR_LIGHT_GRAY = 4, new = function() return {free = function() end} end})
-- Every piece of text on a widget and everything inside it.
function H.widgetText(widget, seen, out)
    seen, out = seen or {}, out or {}
    if type(widget) ~= "table" or seen[widget] then return out end
    seen[widget] = true
    for key, value in pairs(widget) do
        if (key == "text" or key == "title" or key == "subtitle") and type(value) == "string" then out[#out + 1] = value
        elseif type(value) == "table" and key ~= "show_parent" then H.widgetText(value, seen, out) end
    end
    return out
end
-- Bytes on disk, or 0 when the file is missing: what a cached file's existence really means.
function H.fileSize(path)
    local file = io.open(path, "rb")
    if not file then return 0 end
    local size = file:seek("end") or 0
    file:close()
    return size
end
-- Whether a widget tree holds this text anywhere.
function H.findText(widget, needle)
    for _, text in ipairs(H.widgetText(widget)) do
        if text:find(needle, 1, true) then return true end
    end
    return false
end
-- The first value stored under this key anywhere in a widget tree (a cover file, say).
function H.findValue(widget, key, seen)
    seen = seen or {}
    if type(widget) ~= "table" or seen[widget] then return nil end
    seen[widget] = true
    if widget[key] ~= nil then return widget[key] end
    for name, value in pairs(widget) do
        if type(value) == "table" and name ~= "show_parent" then
            local found = H.findValue(value, key, seen)
            if found ~= nil then return found end
        end
    end
end
-- The first button (anything with a text and a callback) with this text.
function H.findButton(widget, text, seen)
    seen = seen or {}
    if type(widget) ~= "table" or seen[widget] then return nil end
    seen[widget] = true
    local icons = {Settings = "appbar.settings", Files = "appbar.filebrowser", Refresh = "cre.render.reload", ["Search books"] = "appbar.search"}
    if (widget.text == text or widget.icon == icons[text] and icons[text] ~= nil) and widget.callback then return widget end
    for key, value in pairs(widget) do
        if type(value) == "table" and key ~= "show_parent" then
            local found = H.findButton(value, text, seen)
            if found then return found end
        end
    end
end
-- Behaves like KOReader's list where it matters here: choosing a row runs it and then calls close_callback.
local Menu = {}
Menu.__index = Menu
function Menu:new(fields) fields.is_menu = true; return setmetatable(fields, self) end
function Menu:onMenuChoice(item) if item.callback then item.callback() end return true end
function Menu:updateItems() self.updates = (self.updates or 0) + 1 end
function Menu:switchItemTable(title, items) self.item_table = items; self.switched = true end
function Menu:onMenuSelect(item)
    if item.sub_item_table == nil then
        if item.select_enabled == false then return true end
        if item.select_enabled_func and not item.select_enabled_func() then return true end
        self:onMenuChoice(item)
        -- Real KOReader honours keep_menu_open: the row acts in place and the page stays up.
        if not item.keep_menu_open and self.close_callback then self.close_callback() end
    else
        self.item_table_stack = self.item_table_stack or {}
        table.insert(self.item_table_stack, self.item_table)
        self:switchItemTable(item.text, item.sub_item_table)
    end
    return true
end
preload("ui/widget/menu", Menu)
preload("ui/event", {new = function(_, name, ...) return {name = name, args = {...}} end})
preload("gettext", function(text) return text end)
preload("logger", {
    warn = function(...) H.logs[#H.logs + 1] = {"warn", ...} end,
    err = function(...) H.logs[#H.logs + 1] = {"err", ...} end,
    info = function() end, dbg = function() end,
})
preload("ui/network/manager", {
    online = true,
    isConnected = function(self) return self.online end,
    willRerunWhenConnected = function() return false end,
})
preload("ui/uimanager", {
    show = function(_, widget) H.shown[#H.shown + 1] = widget end,
    close = function(_, widget) H.closed[#H.closed + 1] = widget; if widget.onCloseWidget then widget:onCloseWidget() end end,
    askForRestart = function(_, message) H.restart = message end,
    -- KOReader's own restart entry point: quit(85), which the launcher turns into a restart.
    restartKOReader = function() H.restarted = true end,
    nextTick = function(_, fn) fn() end,
    scheduleIn = function(_, seconds, fn) H.scheduled[#H.scheduled + 1] = {seconds = seconds, fn = fn} end,
    unschedule = function(_, fn)
        for index = #H.scheduled, 1, -1 do if H.scheduled[index].fn == fn then table.remove(H.scheduled, index) end end
    end,
    setDirty = function() end,
})
preload("ltn12", {
    sink = {
        table = function(target) return function(chunk) if chunk then target[#target + 1] = chunk end return 1 end end,
        -- Like LuaSocket's: writes every chunk and closes the file when the data ends.
        file = function(handle) return function(chunk) if chunk then handle:write(chunk) else handle:close() end return 1 end end,
    },
    source = {string = function(text) local sent; return function() if sent then return nil end sent = true return text end end},
})
preload("rapidjson", {encode = encode, decode = decode})
preload("socketutil", {
    LARGE_BLOCK_TIMEOUT = 10, LARGE_TOTAL_TIMEOUT = 30,
    set_timeout = function(_, block, total) H.timeouts[#H.timeouts + 1] = {block = block, total = total} end,
    reset_timeout = function() end,
})
preload("util", {
    partialMD5 = function(path) return "md5:" .. path end,
    pathExists = function(path) local f = io.open(path, "r"); if f then f:close() return true end return false end,
    makePath = function(path) return os.execute("mkdir -p '" .. path .. "'") end,
    splitToChars = function(text) local out = {} for ch in text:gmatch("[%z\1-\127\194-\244][\128-\191]*") do out[#out + 1] = ch end return out end,
    tableDeepCopy = function(t) local function copy(v) if type(v) ~= "table" then return v end local c = {} for k, x in pairs(v) do c[k] = copy(x) end return c end return copy(t) end,
})
preload("ffi/sha2", {md5 = function(text) return "h" .. #text end, sha256 = function(text) return H.sum(text) end})
preload("socket.http", {request = function(req)
    H.requests[#H.requests + 1] = {url = req.url, method = req.method, at = os.time(), body = req.source and req.source() or nil, headers = req.headers}
    local code, body = H.respond(req)
    if code == nil then return nil, body end -- LuaSocket's way of saying it could not connect: nil and the reason
    if body then H.sink_result, H.sink_error = req.sink(body) end
    req.sink(nil)
    return 1, code
end})
-- KOReader's two menus: each builds its list of top-bar tabs in setUpdateItemTable.
H.menus = {}
local function menuClass(name, tabs)
    return function()
        local class = H.menus[name]
        if not class then
            class = {}
            class.setUpdateItemTable = function(self)
                self.tab_item_table = {}
                for _, icon in ipairs(tabs) do self.tab_item_table[#self.tab_item_table + 1] = {icon = icon, {text = icon .. " item"}} end
            end
            H.menus[name] = class
        end
        return class
    end
end
H.menuTabs = {
    ["apps/filemanager/filemanagermenu"] = {"appbar.filebrowser", "appbar.settings", "appbar.tools", "appbar.search", "appbar.menu"},
    ["apps/reader/modules/readermenu"] = {"appbar.navigation", "appbar.typeset", "appbar.settings", "appbar.tools", "appbar.search", "appbar.menu"},
}
function H.installMenus() for module, tabs in pairs(H.menuTabs) do package.preload[module] = menuClass(module, tabs) end end
H.installMenus()
preload("datastorage", {getDataDir = function() return H.datadir end})
preload("libs/libkoreader-lfs", {
    attributes = function(path, what)
        local ran = os.execute("test -d '" .. path .. "'")
        if ran == true or ran == 0 then return what == "mode" and "directory" or {mode = "directory"} end
        local f = io.open(path, "rb")
        if not f then return nil end
        local size = f:seek("end")
        f:close()
        return what == "size" and size or {mode = "file", size = size}
    end,
    -- Real lfs answers true on success and nil plus a reason on failure; a plain os.execute number is truthy
    -- either way, which would hide a refused mkdir from every caller that checks it.
    mkdir = function(path) local code = os.execute("mkdir '" .. path .. "' 2>/dev/null"); if code == true or code == 0 then return true end return nil, "exists" end,
    rmdir = function(path) local code = os.execute("rmdir '" .. path .. "' 2>/dev/null"); if code == true or code == 0 then return true end return nil, "not empty" end,
    dir = function(path)
        local names, handle = {}, io.popen("ls -a '" .. path .. "' 2>/dev/null")
        if not handle then return function() return nil end end
        for name in handle:lines() do names[#names + 1] = name end
        handle:close()
        local index = 0
        return function() index = index + 1; return names[index] end
    end,
})
preload("mochil_config", {server_url = "https://books.test", token = "t0ken"})

-- A stand-in checksum: equal text gives equal sums, which is all the plugin compares.
function H.sum(text)
    local total = 0
    for index = 1, #text do total = (total * 31 + text:byte(index)) % 4294967291 end
    return string.format("%08x:%d", total, #text)
end

function H.core_source_for_loader()
    return H.read((arg[0]:gsub("[^/]*$", "")) .. "../core.lua")
end
function H.read(path) local f = io.open(path, "rb"); if not f then return nil end local d = f:read("a"); f:close(); return d end
function H.write(path, data) local f = assert(io.open(path, "wb")); f:write(data); f:close() end

function H.reset()
    -- Each test starts a separate simulated process; tests of recovery set this explicitly.
    G_reader_settings:saveSetting("mochil_page_pending", false)
    H.restart = nil
    for module in pairs(H.menuTabs) do package.loaded[module] = nil; H.menus[module] = nil end
    H.hooked_reset = true
    H.cleanTemp()
    H.datadir = H.tmpdir()
    freshHome()
    require("mochil_config").server_url = "https://books.test"
    H.logs, H.shown, H.scheduled, H.requests, H.timeouts, H.events, H.closed, H.sink_result, H.sink_error = {}, {}, {}, {}, {}, {}, {}, nil, nil
    H.restarted, H.restart = false, nil
    H.opened = {}
    H.respond = function() return 200, "{}" end
    H.registered_actions = {}
    H.notified = {}
    H.highlight_dialog_buttons = {}
end

local store = {}
_G.G_reader_settings = {
    readSetting = function(_, key) return store[key] end,
    saveSetting = function(_, key, value) store[key] = value end,
}

-- The page a bookmark item or an xpointer lands on, the way ReaderBookmark:n getDogearBookmarkIndex compares them.
function H.pageOf(document, value)
    if type(value) == "number" then return value end
    if type(value) ~= "string" then return nil end
    local ok, page = pcall(document.getPageFromXPointer, document, value)
    return ok and tonumber(page) or nil
end

-- A reader with a reflowable book open. Pass overrides to break parts of it.
function H.newPlugin(overrides)
    local settings = {}
    local ui = {
        document = {file = "/books/a.epub", info = {has_pages = false}, getXPointer = function() return "/body/DocFragment[1]/p[1]" end,
            getTextFromXPointers = function() return "some words" end, isXPointerInDocument = function() return true end,
            getPageFromXPointer = function(_, xpointer) return tonumber(tostring(xpointer):match("DocFragment%[(%d+)%]")) or 1 end},
        doc_settings = {readSetting = function(_, k) return settings[k] end, saveSetting = function(_, k, v) settings[k] = v end},
        doc_props = {display_title = "A Book", authors = "An Author"},
        rolling = {getLastProgress = function() return "/body/DocFragment[3]/p[2]" end, getLastPercent = function() return 0.25 end},
        annotation = {annotations = {}, addItem = function(self, item) self.annotations[#self.annotations + 1] = item end},
        toc = {getTocTitleByPage = function() return "Chapter" end},
        menu = {registerToMainMenu = function() end},
        highlight = {addToHighlightDialog = function(self, id, fn) H.highlight_dialog_buttons[id] = fn end},
        handleEvent = function(_, event) H.events[#H.events + 1] = event end,
    }
    -- Behaves like ReaderBookmark: toggling a page adds the bookmark when it is missing and removes it when it is
    -- there, and the item it creates has a page, a chapter and no drawer. Built after the reader table, because a
    -- constructor cannot see the local it is being assigned to.
    ui.bookmark = {
        toggles = 0,
        toggleBookmark = function(self, page)
            self.toggles = self.toggles + 1
            local annotations = ui.annotation.annotations
            for index, item in ipairs(annotations) do
                if not item.drawer and item.page ~= nil and H.pageOf(ui.document, item.page) == page then
                    table.remove(annotations, index)
                    return
                end
            end
            annotations[#annotations + 1] = {page = page, chapter = "Chapter", text = "in Chapter"}
        end,
        isPageBookmarked = function(self, page)
            for _, item in ipairs(ui.annotation.annotations) do
                if not item.drawer and item.page ~= nil and H.pageOf(ui.document, item.page) == H.pageOf(ui.document, page) then return true end
            end
            return false
        end,
    }
    for key, value in pairs(overrides or {}) do ui[key] = value end
    local here = arg[0]:gsub("[^/]*$", "")
    -- The plugin folder as KOReader would have it: the core file the updater replaces.
    H.dir = H.tmpdir()
    H.dir = H.dir .. "/"
    H.core_source = H.read(here .. "../core.lua")
    H.write(H.dir .. "mochil_core.lua", H.core_source)
    local loader = {api = 1, dir = H.dir, core_file = "mochil_core.lua", backup_file = "mochil_core.lua.bak"}
    local Mochil = assert(load(H.core_source, "core.lua"))(loader)
    local plugin = Mochil:new{ui = ui}
    return plugin, ui, settings
end

return H
