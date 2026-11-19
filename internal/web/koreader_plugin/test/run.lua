package.path = (arg[0]:gsub("[^/]*$", "")) .. "?.lua;" .. package.path
local H = require("harness")
local failures, passed = {}, 0
local function test(name, fn)
    H.reset()
    if H.reset_fixtures then H.reset_fixtures() end
    local ok, err = xpcall(fn, debug.traceback)
    if ok then passed = passed + 1 else failures[#failures + 1] = name .. "\n" .. tostring(err) end
end
local function eq(actual, expected, message)
    if actual ~= expected then error((message or "values differ") .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual), 2) end
end
local function fired(pattern) for _, line in ipairs(H.requests) do if line.url:find(pattern, 1, true) then return true end end return false end

test("an oversized response fails instead of growing a table until the reader dies", function()
    local plugin = H.newPlugin()
    -- Bigger than the 8 MB JSON ceiling: a misbehaving or hostile server must not be
    -- able to make the plugin hold it in memory.
    H.respond = function() return 200, string.rep("x", 9 * 1024 * 1024) end
    local result, err = plugin:request("GET", nil, true, "status")
    eq(result, nil)
    eq(err, "too_large")
    eq(plugin.last_error, "response too large")
end)

test("a response inside the ceiling is still read normally", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{ok = true} end
    local result, err = plugin:request("GET", nil, true, "status")
    eq(err, nil)
    eq(result.ok, true)
end)

test("an update offers a restart that actually restarts KOReader", function()
    local plugin = H.newPlugin()
    H.restarted = false
    plugin:offerRestart("Mochil Sync was updated to version 1.26.0.")
    local box = H.shown[#H.shown]
    assert(box and box.ok_callback, "a confirmation with a restart button is shown")
    assert(box.ok_text and box.ok_text ~= "", "the button says what it does")
    box.ok_callback()
    assert(H.restarted, "pressing Restart now restarts KOReader through restartKOReader")
end)

test("a KOReader without restartKOReader still asks its own way", function()
    local plugin = H.newPlugin()
    local ui = require("ui/uimanager")
    local saved = ui.restartKOReader
    ui.restartKOReader = nil
    H.restart = nil
    plugin:offerRestart("updated")
    eq(H.restart, "updated", "it falls back to UIManager:askForRestart")
    ui.restartKOReader = saved
end)

test("pushing sends the position with the book's identity", function()
    local plugin = H.newPlugin()
    plugin:push(true)
    eq(#H.requests, 1)
    eq(H.requests[1].method, "PUT")
    local body = H.decode(H.requests[1].body)
    eq(body.progress, "/body/DocFragment[3]/p[2]")
    eq(body.title, "A Book")
    eq(body.device, "TestReader")
end)

test("an unmatched position is not acknowledged and is retried", function()
    local plugin, _, settings = H.newPlugin()
    H.respond = function() return 200, H.encode{saved=false, linked=false} end
    plugin:push(true)
    eq(plugin.last_sent_position, nil)
    eq(settings.mochil_pushed_at, nil)
    eq(settings.mochil_seen_at, nil)
    assert(not H.shown[#H.shown].text:find("Progress sent", 1, true))
    plugin:push(false)
    eq(#H.requests, 2)
end)

test("pulling a newer position moves the reader there", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{position = {koreader_locator = "/body/DocFragment[7]/p[1]", updated_at = 50, fraction = 0.6}} end
    plugin:pull(true)
    eq(H.events[1] and H.events[1].name, "GotoXPointer")
    eq(H.events[1].args[1], "/body/DocFragment[7]/p[1]")
end)

test("a highlight deleted elsewhere is removed through KOReader's own annotation list", function()
    local removed
    local plugin, ui, settings = H.newPlugin()
    ui.annotation.annotations = {{mochil_id = "r1", drawer = "lighten", text = "x", pos0 = "a", pos1 = "b"}}
    settings.mochil_annotation_snapshot = {r1={revision=1, signature=plugin:localAnnotations().r1._signature}}
    ui.annotation.removeItemByIndex = function(self, index) removed = index; table.remove(self.annotations, index) end
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 2, applied = {{id = "r1", deleted = true, revision = 2}}} end
        return 200, H.encode{revision = 2, applied = {}, conflicts = {}}
    end
    plugin:syncAnnotations(true)
    eq(removed, 1, "removeItemByIndex was not used")
    eq(#ui.annotation.annotations, 0)
end)

test("a reader without removeItemByIndex still loses the deleted highlight", function()
    local plugin, ui, settings = H.newPlugin()
    ui.annotation.annotations = {{mochil_id = "r1", drawer = "lighten", text = "x", pos0 = "a", pos1 = "b"}}
    settings.mochil_annotation_snapshot = {r1={revision=1, signature=plugin:localAnnotations().r1._signature}}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 2, applied = {{id = "r1", deleted = true, revision = 2}}} end
        return 200, H.encode{revision = 2, applied = {}, conflicts = {}}
    end
    plugin:syncAnnotations(true)
    eq(#ui.annotation.annotations, 0)
end)

test("a remote edit cannot overwrite an unsent local note", function()
    local plugin, ui, settings = H.newPlugin()
    local item = {mochil_id="r1", mochil_revision=1, drawer="lighten", text="quote", note="old", pos0="a", pos1="b", datetime="2026-01-01 10:00:00"}
    ui.annotation.annotations = {item}
    local original = plugin:localAnnotations().r1._signature
    settings.mochil_annotation_snapshot = {r1={revision=1, signature=original}}
    settings.mochil_annotation_revision = 1
    settings.mochil_annotation_revision_cursor_version = 1
    item.note = "local edit"
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={{id="r1", revision=2, quote="quote", note="remote edit", locators={koreader=H.encode{pos0="a", pos1="b"}}}}} end
        return 200, H.encode{revision=2, applied={}, conflicts={{id="r1"}}}
    end
    plugin:syncAnnotations(true)
    eq(item.note, "local edit", "remote edit destroyed the local note")
    local sent = H.decode(H.requests[2].body).changes[1]
    eq(sent.annotation.note, "local edit")
    eq(sent.base_revision, 1, "conflict must use the last synchronized revision")
    eq(settings.mochil_annotation_revision, 1, "the revision cursor must not skip a conflicted remote edit")
end)

test("a remote match with a different id cannot overwrite an unsent local note", function()
    local plugin, ui = H.newPlugin()
    local item = {drawer="lighten", text="quote", note="local", pos0="a", pos1="b"}
    ui.annotation.annotations = {item}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={{id="r1", revision=2, quote="quote", note="remote", locators={koreader=H.encode{pos0="a", pos1="b"}}}}} end
        return 200, H.encode{revision=2, applied={}, conflicts={}}
    end
    plugin:syncAnnotations(true)
    eq(item.note, "local", "remote match destroyed an unsent local note")
    eq(H.decode(H.requests[2].body).changes[1].annotation.note, "local")
end)

test("a conflict during an interactive sync shows a blocking message", function()
    local plugin, ui = H.newPlugin()
    local item = {drawer="lighten", text="quote", note="local edit", pos0="a", pos1="b", datetime="2026-01-01 10:00:00"}
    ui.annotation.annotations = {item}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={}} end
        local submitted = H.decode(H.requests[#H.requests].body).changes[1].annotation
        return 200, H.encode{revision=3, applied={}, conflicts={{submitted=submitted, current={id=submitted.id, revision=3,
            updated_at=(submitted.updated_at or 0) + 1000, deleted=false, quote="quote", note="remote",
            locators={koreader=H.encode{pos0="/body/DocFragment[1]/p[1]", pos1="/body/DocFragment[1]/p[1]"}}}}}}
    end
    plugin:syncAnnotations(true)
    local foundConflict = false
    for _, widget in ipairs(H.shown) do if widget.text and widget.text:find("changed elsewhere", 1, true) then foundConflict = true end end
    assert(foundConflict, "an interactive sync should show the conflict, not stay silent")
    eq(#H.notified, 0, "an interactive sync must not use the background toast")
end)

test("a conflict during a background sync uses a non-blocking toast, not a modal", function()
    local plugin, ui = H.newPlugin()
    local item = {drawer="lighten", text="quote", note="local edit", pos0="a", pos1="b", datetime="2026-01-01 10:00:00"}
    ui.annotation.annotations = {item}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={}} end
        local submitted = H.decode(H.requests[#H.requests].body).changes[1].annotation
        return 200, H.encode{revision=3, applied={}, conflicts={{submitted=submitted, current={id=submitted.id, revision=3,
            updated_at=(submitted.updated_at or 0) + 1000, deleted=false, quote="quote", note="remote",
            locators={koreader=H.encode{pos0="/body/DocFragment[1]/p[1]", pos1="/body/DocFragment[1]/p[1]"}}}}}}
    end
    plugin:syncAnnotations(false)
    eq(#H.shown, 0, "a background sync's conflict must not pop a blocking InfoMessage")
    assert(#H.notified == 1 and H.notified[1]:find("changed elsewhere", 1, true), "a background sync's conflict should use a non-blocking toast")
end)

-- A conflict with no terminal state was re-sent, and re-announced, on every sync for ever. The newer edit wins once.
test("a conflict is settled once instead of being retried on every sync", function()
    local plugin, ui = H.newPlugin()
    local item = {drawer="lighten", text="quote", note="newer local edit", pos0="a", pos1="b", datetime="2026-06-01 10:00:00"}
    ui.annotation.annotations = {item}
    local puts = 0
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={}} end
        puts = puts + 1
        local changes = H.decode(H.requests[#H.requests].body).changes
        local submitted = changes[1].annotation
        if puts == 1 then
            -- The server still holds an older revision of this highlight: a real conflict.
            return 200, H.encode{revision=3, applied={}, conflicts={{submitted=submitted, current={id=submitted.id, revision=3,
                updated_at=(submitted.updated_at or 0) - 60000, deleted=false, quote="quote", note="older remote edit"}}}}
        end
        eq(changes[1].base_revision, 3, "the newer edit must be re-sent against the conflicting revision")
        return 200, H.encode{revision=4, applied={{id=submitted.id, revision=4}}}
    end
    plugin:syncAnnotations(false)
    eq(puts, 2, "the newer edit was not re-sent")
    plugin:syncAnnotations(false)
    eq(puts, 2, "the settled conflict was uploaded again on the next sync")
    eq(#H.notified, 1, "the settled conflict was announced again")
end)

test("a matched remote note cannot overwrite a local note after it was uploaded", function()
    local plugin, ui, settings = H.newPlugin()
    local item = {drawer="lighten", text="quote", note="local", pos0="a", pos1="b"}
    ui.annotation.annotations = {item}
    local local_id, annotation = next(plugin:localAnnotations())
    settings.mochil_annotation_snapshot = {[local_id]={revision=1, signature=annotation._signature}}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={{id="r1", revision=2, quote="quote", note="remote", locators={koreader=H.encode{pos0="a", pos1="b"}}}}} end
        return 200, H.encode{revision=2, applied={}, conflicts={}}
    end
    plugin:syncAnnotations(true)
    eq(item.note, "local", "a different remote note overwrote the uploaded local note")
end)

test("a snapshotless local edit survives a conflicting remote edit", function()
    local plugin, ui = H.newPlugin()
    local item = {mochil_id="r1", mochil_revision=1, drawer="lighten", text="quote", note="local", pos0="a", pos1="b"}
    ui.annotation.annotations = {item}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={{id="r1", revision=2, quote="quote", note="remote", locators={koreader=H.encode{pos0="a", pos1="b"}}}}} end
        return 200, H.encode{revision=2, applied={}, conflicts={{id="r1"}}}
    end
    plugin:syncAnnotations(true)
    eq(item.note, "local")
    eq(H.decode(H.requests[2].body).changes[1].base_revision, 0)
end)

test("a remote edit does not resurrect a locally deleted highlight", function()
    local plugin, ui, settings = H.newPlugin()
    local item = {mochil_id="r1", mochil_revision=1, drawer="lighten", text="quote", pos0="a", pos1="b", datetime="2026-01-01 10:00:00"}
    ui.annotation.annotations = {item}
    settings.mochil_annotation_snapshot = {r1={revision=1, signature=plugin:localAnnotations().r1._signature}}
    ui.annotation.annotations = {}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision=2, applied={{id="r1", revision=2, quote="quote", note="remote", locators={koreader=H.encode{pos0="a", pos1="b"}}}}} end
        return 200, H.encode{revision=2, applied={}, conflicts={{id="r1"}}}
    end
    plugin:syncAnnotations(true)
    eq(#ui.annotation.annotations, 0, "remote edit resurrected local deletion")
    plugin:syncAnnotations(true)
    eq(#ui.annotation.annotations, 0)
    local change = H.decode(H.requests[#H.requests].body).changes[1]
    eq(change.annotation.deleted, true)
    eq(change.base_revision, 1)
end)

test("an error inside the plugin never reaches KOReader", function()
    local broken = {getLastProgress = function() error("boom") end, getLastPercent = function() return 0 end}
    local plugin = H.newPlugin({rolling = broken})
    H.respond = function() return 200, H.encode{position = {koreader_locator = "/body/DocFragment[7]/p[1]", updated_at = 50, fraction = 0.6}} end
    -- Each of these reaches the failing call from a different entry point, the way KOReader would.
    plugin:onCloseDocument()
    plugin:onSuspend()
    plugin:setSyncPolicy("page")
    plugin:onPageUpdate()
    plugin:onPosUpdate()
    plugin:onReaderReady()
    for _, task in ipairs({table.unpack(H.scheduled)}) do task.fn() end
    plugin:pull(false)
    plugin:push(false)
    plugin:syncAnnotations(false)
    plugin:setSyncPolicy("close")
    local logged = false
    for _, line in ipairs(H.logs) do if line[1] == "err" and tostring(line[2]):find("boom", 1, true) then logged = true end end
    eq(logged, true, "the failure should be logged")
end)

test("an interactive action that fails tells the reader instead of crashing", function()
    local plugin = H.newPlugin({rolling = {getLastProgress = function() error("boom") end, getLastPercent = function() return 0 end}})
    plugin:push(true)
    eq(#H.shown, 1)
end)

test("a garbled server reply is survived", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "not json at all" end
    plugin:pull(true)
    plugin:syncAnnotations(true)
    H.respond = function() return 200, H.encode{applied = "oops", revision = "x"} end
    plugin:syncAnnotations(true)
end)

test("closing a book stops asking once its time budget is spent", function()
    local real, now = os.time, 1000
    os.time = function(t) if t then return real(t) end return now end
    local plugin, ui = H.newPlugin()
    ui.annotation.annotations = {{drawer = "lighten", text = "a quote", pos0 = "/body/DocFragment[1]/p[1]", pos1 = "/body/DocFragment[1]/p[2]", datetime = "2026-01-01 10:00:00"}}
    H.respond = function(req) now = now + 5; return 200, H.encode{revision = 1, applied = {}} end
    plugin:onCloseDocument()
    os.time = real
    assert(#H.requests >= 2, "the position and the highlight check should both be tried")
    for _, request in ipairs(H.requests) do assert(request.at < 1006, "a request started after the budget: " .. request.method .. " at " .. request.at) end
    for _, t in ipairs(H.timeouts) do assert(t.total <= 6, "a wait longer than the budget was allowed") end
end)

test("annotation sync requests only changes after the saved server revision", function()
    local plugin, _, settings = H.newPlugin()
    settings.mochil_annotation_revision = 7
    settings.mochil_annotation_revision_cursor_version = 1
    plugin:syncAnnotations(false)
    eq(H.requests[1].url:find("?since=7", 1, true) ~= nil, true, "the saved revision was not sent to the server")
end)

test("legacy saved revisions trigger one full annotation sync before using the cursor", function()
    local plugin, _, settings = H.newPlugin()
    settings.mochil_annotation_revision = 7
    H.respond = function() return 200, H.encode{revision=7, applied={}} end
    plugin:syncAnnotations(false)
    eq(H.requests[1].url:find("?since=0", 1, true) ~= nil, true, "legacy revision must not hide annotations from the first cursor-based sync")
    eq(settings.mochil_annotation_revision, 7)
    eq(settings.mochil_annotation_revision_cursor_version, 1)
end)

test("an unresolved remote annotation does not advance the saved revision cursor", function()
    local plugin, _, settings = H.newPlugin()
    settings.mochil_annotation_revision = 7
    settings.mochil_annotation_revision_cursor_version = 1
    H.respond = function(req)
        if req.method == "GET" then
            return 200, H.encode{revision=8, applied={{id="remote1", revision=8, quote="not present here", locators={koreader="invalid-json"}}}}
        end
        return 200, H.encode{revision=8, applied={}, conflicts={}}
    end
    plugin:syncAnnotations(false)
    eq(settings.mochil_annotation_revision, 7, "an annotation the reader could not import must remain eligible for retry")
end)

test("going to sleep only sends the position", function()
    local plugin = H.newPlugin()
    plugin:onSuspend()
    eq(fired("/progress/"), true)
    eq(fired("/annotations/"), false)
end)

test("manual mode blocks queued and open-time automatic sync", function()
    local plugin = H.newPlugin()
    plugin:onAnnotationsModified()
    plugin:onReaderReady()
    plugin:setSyncPolicy("manual")
    for _, task in ipairs({table.unpack(H.scheduled)}) do task.fn() end
    plugin:onAnnotationsModified()
    for _, task in ipairs({table.unpack(H.scheduled)}) do task.fn() end
    eq(fired("/progress/"), false, "manual mode sent automatic progress")
    eq(fired("/annotations/"), false, "manual mode sent automatic highlights")
end)

test("manual sync mode never syncs in the background", function()
    local plugin = H.newPlugin()
    plugin:setSyncPolicy("manual")
    plugin:onCloseDocument()
    plugin:onSuspend()
    eq(#H.requests, 0)
    plugin:setSyncPolicy("close")
end)

-- ===== updating the plugin from inside KOReader =====
local function offer(version, overrides)
    local core = H.core_source:gsub('Mochil.release = "[%d.]+"', 'Mochil.release = "' .. version .. '"', 1)
    local manifest = {version = version, sha256 = H.sum(core), size = #core, loader_api = 1}
    for key, value in pairs(overrides or {}) do manifest[key] = value end
    H.respond = function(req)
        if req.url:find("/plugin/manifest", 1, true) then return 200, H.encode(manifest) end
        if req.url:find("/plugin/core.lua", 1, true) then return 200, core end
        return 200, "{}"
    end
    return core, manifest
end
local function accept(plugin)
    local box = H.shown[#H.shown]
    assert(box and box.ok_callback, "no update offer was shown")
    box.ok_callback()
end

test("a newer plugin is offered and installed, keeping the old one as a fallback", function()
    local plugin = H.newPlugin()
    local core = offer("9.9.9")
    plugin:checkForUpdate(true)
    accept(plugin)
    eq(H.read(H.dir .. "mochil_core.lua"), core, "the new core was not written")
    eq(H.read(H.dir .. "mochil_core.lua.bak"), H.core_source, "the old core was not kept")
    local restartBox = H.shown[#H.shown]
    assert(restartBox and restartBox.ok_callback, "no restart offer was shown after installing")
    restartBox.ok_callback()
    assert(H.restarted, "Restart now did not restart KOReader")
    eq(G_reader_settings:readSetting("mochil_update_available"), false)
end)

test("the pairing file is never touched by an update", function()
    local plugin = H.newPlugin()
    H.write(H.dir .. "mochil_config.lua", "return { token = 'keep-me' }")
    offer("9.9.9")
    plugin:checkForUpdate(true); accept(plugin)
    eq(H.read(H.dir .. "mochil_config.lua"), "return { token = 'keep-me' }")
end)

test("an oversized updater response is refused by the response sink", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "oversized" end
    local response, err = plugin:fetchRaw("/api/koreader/v1/plugin/core.lua", 4)
    eq(response, nil)
    eq(err, "too_large")
    eq(H.sink_error, "too_large", "the receiver must stop accepting bytes at the size limit")
end)

test("a download that does not match the checksum is refused", function()
    local plugin = H.newPlugin()
    offer("9.9.9", {sha256 = "00000000:1"})
    plugin:checkForUpdate(true); accept(plugin)
    eq(H.read(H.dir .. "mochil_core.lua"), H.core_source, "the core must stay as it was")
    eq(H.read(H.dir .. "mochil_core.lua.bak"), nil)
    eq(H.restart, nil)
end)

test("a download that is not valid Lua is refused", function()
    local plugin = H.newPlugin()
    local broken = "this is not lua ((("
    H.respond = function(req)
        if req.url:find("/plugin/manifest", 1, true) then return 200, H.encode{version = "9.9.9", sha256 = H.sum(broken), size = #broken} end
        return 200, broken
    end
    plugin:checkForUpdate(true); accept(plugin)
    eq(H.read(H.dir .. "mochil_core.lua"), H.core_source)
end)

test("an update that cannot be written leaves the plugin as it was", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    os.execute("chmod a-w " .. H.dir); os.execute("rm -f " .. H.dir .. "mochil_core.lua.new")
    local probe = io.open(H.dir .. "probe", "w")
    if probe then probe:close(); os.remove(H.dir .. "probe"); os.execute("chmod u+w " .. H.dir); return end -- running as root: cannot make a read-only folder
    plugin:checkForUpdate(true); accept(plugin)
    os.execute("chmod u+w " .. H.dir)
    eq(H.read(H.dir .. "mochil_core.lua"), H.core_source)
end)

test("the same or an older version is not offered", function()
    local plugin = H.newPlugin()
    offer("1.2.0")
    plugin:checkForUpdate(true)
    eq(H.shown[#H.shown].ok_callback, nil, "no install should be offered")
    offer("0.9.0")
    plugin:checkForUpdate(true)
    eq(H.shown[#H.shown].ok_callback, nil)
end)

test("an update that needs a newer loader asks for a fresh install", function()
    local plugin = H.newPlugin()
    offer("9.9.9", {loader_api = 2})
    plugin:checkForUpdate(true)
    eq(H.shown[#H.shown].ok_callback, nil)
    eq(H.read(H.dir .. "mochil_core.lua"), H.core_source)
end)

test("code is only downloaded over https", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    local config = require("mochil_config")
    config.server_url = "http://books.example.com"
    plugin:checkForUpdate(true)
    eq(#H.requests, 0)
    local trusted = plugin.updatesAllowed
    eq(trusted("http://172.20.0.5"), false); eq(trusted("http://172.40.0.5"), false); eq(trusted("ftp://x"), false)
end)

test("executable updates are never fetched over plain http", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    local config = require("mochil_config")
    config.server_url = "http://192.0.2.10:8099"
    plugin:checkForUpdate(true)
    eq(#H.requests, 0, "a local network does not authenticate executable code")
    eq(plugin.updatesAllowed("http://localhost"), false)
    eq(plugin.updatesAllowed("https://books.example.com"), true)
end)

test("update requests disable redirects", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    local respond = H.respond
    H.respond = function(req)
        eq(req.redirect, false, "update request can be redirected to plaintext or another host")
        return respond(req)
    end
    plugin:checkForUpdate(true)
    assert(#H.requests > 0)
    assert(H.shown[#H.shown].ok_callback, "update manifest should be accepted")
end)

test("an automatic check runs at most once a day and ignores the old off switch", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    G_reader_settings:saveSetting("mochil_update_checked_at", 0)
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 1)
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 1, "the second check the same day should not ask again")
    G_reader_settings:saveSetting("mochil_update_checked_at", 0)
    G_reader_settings:saveSetting("mochil_auto_update_check", false)
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 2, "a stored 'off' from an older version no longer stops the daily check")
    G_reader_settings:saveSetting("mochil_auto_update_check", nil)
end)

test("an automatic check that fails waits hours before trying again", function()
    local plugin = H.newPlugin()
    H.respond = function() return 500, "" end
    G_reader_settings:saveSetting("mochil_update_checked_at", 0)
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 1)
    plugin:maybeCheckForUpdate()
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 1, "every book opened must not retry a failing check")
    -- A failed attempt is stored as "18 hours ago", so the next try is due 6 hours later; simulate 7 hours passing.
    G_reader_settings:saveSetting("mochil_update_checked_at", os.time() - 64800 - 7 * 3600)
    plugin:maybeCheckForUpdate()
    eq(#H.requests, 2, "after the back-off it tries again")
end)

test("an automatic check announces an update quietly, once", function()
    local plugin = H.newPlugin()
    offer("9.9.9")
    G_reader_settings:saveSetting("mochil_update_checked_at", 0)
    G_reader_settings:saveSetting("mochil_update_announced", nil)
    plugin:maybeCheckForUpdate()
    eq(#H.shown, 1); eq(H.shown[1].ok_callback, nil, "it must not interrupt reading with a question")
    G_reader_settings:saveSetting("mochil_update_checked_at", 0)
    plugin:maybeCheckForUpdate()
    eq(#H.shown, 1, "the same version is announced only once")
end)

test("the version comparison orders releases", function()
    local newer = H.newPlugin().isNewerVersion
    eq(newer("1.10.0", "1.9.9"), true); eq(newer("1.2.1", "1.2.0"), true)
    eq(newer("1.2.0", "1.2.0"), false); eq(newer("1.1.9", "1.2.0"), false); eq(newer("junk", "1.2.0"), false)
end)

-- ===== the loader that starts the plugin =====
local function loaderFolder(core, backup)
    local dir = H.tmpdir(); dir = dir .. "/"
    H.write(dir .. "main.lua", H.read(arg[0]:gsub("[^/]*$", "") .. "../main.lua"))
    if core then H.write(dir .. "mochil_core.lua", core) end
    if backup then H.write(dir .. "mochil_core.lua.bak", backup) end
    return dir
end
local function startLoader(dir)
    local chunk = assert(loadfile(dir .. "main.lua"))
    return chunk()
end

test("after an update and a restart, the loader runs the new version", function()
    local plugin = H.newPlugin()
    local core = offer("9.9.9")
    plugin:checkForUpdate(true)
    accept(plugin)
    H.write(H.dir .. "main.lua", H.read(arg[0]:gsub("[^/]*$", "") .. "../main.lua"))
    local class = startLoader(H.dir)
    eq(class.release, "9.9.9", "the restarted plugin is still the old version")
end)

test("when the loader had to fall back, the plugin says why", function()
    local dir = loaderFolder("this is not lua (((", H.core_source_for_loader())
    local class = startLoader(dir)
    assert(class.fallback_error and #class.fallback_error > 0, "the reason for the fallback was not kept")
    local plugin = class:new{ui = {menu = {registerToMainMenu = function() end}}}
    plugin:showInstallDetails()
    assert(H.shown[#H.shown].text:find(class.fallback_error, 1, true), "the details should show the reason")
end)

test("an update that is installed but not running is visible in the menu", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_installed_version", "9.9.9")
    local entries = plugin:menuEntries()
    local text
    for _, entry in ipairs(entries) do if entry.text_func then local t = entry.text_func(); if t:find("not running", 1, true) then text = t end end end
    assert(text, "a pending update that did not take effect should be noticeable")
    G_reader_settings:saveSetting("mochil_installed_version", nil)
end)

test("the menu offers highlight export actions, book-scoped ones gated on an open book", function()
    local plugin = H.newPlugin()
    local entries = plugin:menuEntries()
    local function find(label) for _, entry in ipairs(entries) do if entry.text == label then return entry end end end
    local notes = find("Export this book's highlights to Notes")
    local backup = find("Back up all my highlights")
    assert(notes and notes.callback, "a menu entry must export this book's highlights to Notes")
    assert(notes.enabled_func and notes.enabled_func() == true, "exporting to Notes needs an open book")
    assert(backup and backup.callback, "a menu entry must back up all highlights")
    assert(backup.enabled_func == nil, "backing up all highlights needs no open book")
end)

test("push, pull and sync-highlights are registered as gesture-bindable Dispatcher actions", function()
    local plugin = H.newPlugin()
    plugin:onDispatcherRegisterActions()
    local push = H.registered_actions["mochil_push"]
    local pull = H.registered_actions["mochil_pull"]
    local sync = H.registered_actions["mochil_sync_highlights"]
    assert(push and push.event and push.general == true, "push must be a general, gesture-bindable action")
    assert(pull and pull.event and pull.general == true, "pull must be a general, gesture-bindable action")
    assert(sync and sync.event and sync.general == true, "sync highlights must be a general, gesture-bindable action")
end)

test("the Dispatcher-registered events trigger the matching push/pull/sync action", function()
    local plugin = H.newPlugin()
    plugin:onDispatcherRegisterActions()
    plugin["on" .. H.registered_actions["mochil_push"].event](plugin)
    eq(H.requests[#H.requests].method, "PUT", "the push gesture must push")
    H.respond = function() return 200, H.encode{position = {koreader_locator = "/body/DocFragment[1]/p[1]", updated_at = 1, fraction = 0.1}} end
    plugin["on" .. H.registered_actions["mochil_pull"].event](plugin)
    eq(H.events[#H.events] and H.events[#H.events].name, "GotoXPointer", "the pull gesture must pull")
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 1, applied = {}} end
        return 200, H.encode{revision = 1, applied = {}, conflicts = {}}
    end
    plugin["on" .. H.registered_actions["mochil_sync_highlights"].event](plugin)
    assert(fired("/annotations/"), "the sync-highlights gesture must sync annotations")
end)

test("the reader-ready hook adds a 'Sync to Mochil' button to the highlight popup", function()
    local plugin = H.newPlugin()
    plugin:onReaderReady()
    local register = H.highlight_dialog_buttons["mochil_sync_highlight"]
    assert(register, "a highlight dialog button must be registered on reader ready")
    local button = register()
    assert(button and button.text and type(button.callback) == "function", "the registered entry must be a usable button")
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 1, applied = {}} end
        return 200, H.encode{revision = 1, applied = {}, conflicts = {}}
    end
    button.callback()
    assert(fired("/annotations/"), "tapping the button must sync highlights")
end)

test("the highlight popup button is not registered when the reader has no highlight module", function()
    local plugin, ui = H.newPlugin()
    ui.highlight = nil
    plugin:onReaderReady()
    eq(H.highlight_dialog_buttons["mochil_sync_highlight"], nil)
end)


test("the loader starts the plugin from its core", function()
    local class = startLoader(loaderFolder(H.core_source_for_loader()))
    eq(type(class.push), "function")
    eq(class.release ~= nil, true)
end)

test("a broken new core falls back to the previous one and heals itself", function()
    local dir = loaderFolder("this is not lua (((", H.core_source_for_loader())
    local class = startLoader(dir)
    eq(type(class.push), "function", "the previous core should have been used")
    eq(H.read(dir .. "mochil_core.lua"), H.core_source_for_loader(), "the working core was not put back")
end)

test("a core that loads but is not a plugin falls back too", function()
    local dir = loaderFolder("return 42", H.core_source_for_loader())
    eq(type(startLoader(dir).push), "function")
end)

test("with nothing usable the loader offers a reinstall instead of failing", function()
    local class = startLoader(loaderFolder("garbage (((", "more garbage ((("))
    local menu = {}
    class.addToMainMenu({}, menu)
    assert(menu.mochil and menu.mochil.text:find("reinstall", 1, true), "no reinstall entry")
end)

-- ===== the top-bar tab =====
-- KOReader builds its tabs in setUpdateItemTable; the plugin adds one after the search tab once that has run.
local function freshMenus()
    local Mochil = getmetatable(H.newPlugin()).__index
    Mochil.hooked = nil
    for module in pairs(H.menus) do H.menus[module] = nil; package.loaded[module] = nil end
    return Mochil
end
local function openMenu(module)
    local menu = setmetatable({}, {__index = require(module)})
    menu:setUpdateItemTable()
    return menu
end
local function iconsOf(menu) local icons = {} for _, tab in ipairs(menu.tab_item_table) do icons[#icons + 1] = tab.icon end return icons end

test("Mochil gets its own tab right after search, in both menus", function()
    freshMenus()
    local plugin = H.newPlugin(); plugin:init()
    for module, tabs in pairs{["apps/filemanager/filemanagermenu"] = "appbar.search", ["apps/reader/modules/readermenu"] = "appbar.search"} do
        local menu = openMenu(module)
        local icons = iconsOf(menu)
        for position, icon in ipairs(icons) do
            if icon == tabs then eq(menu.tab_item_table[position + 1].mochil_tab, true, module .. ": not next to search") end
        end
        eq(#icons, #H.menuTabs[module] + 1, module .. ": exactly one tab should be added")
    end
end)

test("rebuilding the menu does not add a second tab", function()
    freshMenus()
    local plugin = H.newPlugin(); plugin:init()
    local menu = openMenu("apps/filemanager/filemanagermenu")
    menu:setUpdateItemTable(); menu:setUpdateItemTable()
    local count = 0
    for _, tab in ipairs(menu.tab_item_table) do if tab.mochil_tab then count = count + 1 end end
    eq(count, 1)
    local again = H.newPlugin(); again:init()
    eq(#iconsOf(openMenu("apps/filemanager/filemanagermenu")), #H.menuTabs["apps/filemanager/filemanagermenu"] + 1, "a second plugin instance must not wrap the menu twice")
end)

test("tapping the tab closes KOReader's menu and opens the Mochil page", function()
    freshMenus()
    local plugin = H.newPlugin(); plugin:init()
    H.respond = function() return 200, H.encode{account = "kevin", books = 5758, series = 1191, authors = 1118, collections = 1, continue = {}} end
    local menu = openMenu("apps/filemanager/filemanagermenu")
    local closed = false
    menu.onCloseFileManagerMenu = function() closed = true end
    local tab
    for _, candidate in ipairs(menu.tab_item_table) do if candidate.mochil_tab then tab = candidate end end
    assert(tab and tab.callback, "the tab should do something when tapped")
    tab.callback()
    eq(closed, true, "KOReader's own menu should close first")
    local on_screen = table.concat(H.widgetText(H.shown[#H.shown]), "\n")
    assert(on_screen:find("Continue reading", 1, true) and on_screen:find("kevin", 1, true), "the home page should be on screen, not another:\n" .. on_screen)
end)


test("a KOReader whose menu cannot be wrapped keeps the entries under Tools", function()
    freshMenus()
    package.preload["apps/filemanager/filemanagermenu"] = function() error("no such module") end
    package.loaded["apps/filemanager/filemanagermenu"] = nil
    local plugin = H.newPlugin(); plugin:init()
    local menu = {}
    plugin:addToMainMenu(menu)
    eq(menu.mochil_tools.sorting_hint, "tools")
    assert(#menu.mochil_tools.sub_item_table > 3)
    H.installMenus()
end)

test("a failure while adding the tab leaves KOReader's own menu intact", function()
    freshMenus()
    local plugin = H.newPlugin(); plugin:init()
    package.preload["datastorage"] = function() error("boom") end
    package.loaded["datastorage"] = nil
    local menu = openMenu("apps/filemanager/filemanagermenu")
    assert(#menu.tab_item_table >= #H.menuTabs["apps/filemanager/filemanagermenu"], "KOReader's tabs must all still be there")
    package.preload["datastorage"] = function() return {getDataDir = function() return H.datadir end} end
    package.loaded["datastorage"] = nil -- LuaJIT remembers a failed load until this is cleared
end)

test("the icon is installed, and used only once KOReader can see it", function()
    local plugin = H.newPlugin()
    local first = plugin.prepareIcon()
    eq(first, "book.opened", "the first run has to use a built-in icon")
    local path = H.datadir .. "/icons/mochil.svg"
    assert((H.read(path) or ""):find("<svg", 1, true), "the icon file was not written")
    eq(plugin.prepareIcon(), "mochil", "once the folder existed at start, the Mochil icon is used")
    H.write(path, "stale")
    plugin.prepareIcon()
    assert(H.read(path):find("<svg", 1, true), "an out-of-date icon was not refreshed")
end)

test("an icon that cannot be written falls back instead of failing", function()
    local plugin = H.newPlugin()
    H.datadir = "/nonexistent-folder/deeper"
    eq(plugin.prepareIcon(), "book.opened")
end)

-- ===== saying why a connection failed =====
test("a failed connection says why, in plain words", function()
    local plugin = H.newPlugin()
    for reason, phrase in pairs{["host or service not found"] = "looked up", ["connection refused"] = "refused", ["timeout"] = "in time",
            ["certificate verify failed"] = "secure connection", ["network is unreachable"] = "cannot reach"} do
        H.reset(); H.respond = function() return nil, reason end
        plugin:testConnection()
        local text = H.shown[#H.shown].text
        assert(text:find("books.test", 1, true), "the server address should be shown: " .. text)
        assert(text:find(phrase, 1, true), reason .. " should mention '" .. phrase .. "': " .. text)
        assert(not text:find("Wi%-Fi"), "the message should not blame Wi-Fi when the device may be on mobile data")
    end
end)

test("testing the connection works from the file manager, where no book is open", function()
    local plugin = H.newPlugin({document = false, doc_settings = false})
    plugin.ui.document, plugin.ui.doc_settings, plugin.ui.doc_props, plugin.ui.rolling = nil, nil, nil, nil
    H.respond = function() return 200, H.encode{connected = true, account = "kevin"} end
    plugin:testConnection()
    for _, task in ipairs(H.scheduled) do task.fn() end
    local text = H.shown[#H.shown].text
    assert(text:find("kevin", 1, true), "expected a connection, got: " .. text)
end)

test("a working connection says who it is connected as", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{connected = true, account = "kevin"} end
    plugin:testConnection()
    assert(H.shown[#H.shown].text:find("kevin", 1, true))
end)

test("the plugin loads and works on a Lua 5.1 runtime without table.pack and table.unpack", function()
    local pack, unpack51 = table.pack, table.unpack
    table.pack, table.unpack = nil, nil
    rawset(_G, "unpack", unpack51)
    local ok, err = pcall(function()
        local plugin = H.newPlugin()
        plugin:init()
        plugin:push(true)
        eq(#H.requests, 1, "a push should still be sent")
        local menu = {}
        plugin:addToMainMenu(menu)
        assert(menu.mochil_tools, "the menu entry should still be registered")
    end)
    table.pack, table.unpack = pack, unpack51
    rawset(_G, "unpack", nil)
    assert(ok, err)
end)

local function runOpenSteps()
    local tasks = {}
    for _, task in ipairs(H.scheduled) do if task.seconds >= 1 then tasks[#tasks + 1] = task end end
    table.sort(tasks, function(a, b) return a.seconds < b.seconds end)
    for _, task in ipairs(tasks) do task.fn() end
    return tasks
end

test("opening a book only ever waits a few seconds on the network", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{position = {koreader_locator = "/body/DocFragment[3]/p[2]", updated_at = 1, fraction = 0.25}} end
    plugin:onReaderReady()
    runOpenSteps()
    assert(#H.timeouts >= 1, "the open-time requests should set a timeout")
    for _, t in ipairs(H.timeouts) do
        assert(t.total <= 8, "an open-time request may wait " .. tostring(t.total) .. "s; the screen is frozen meanwhile")
    end
end)

test("opening a book is spread over several short steps, not one long one", function()
    local plugin = H.newPlugin()
    plugin:onReaderReady()
    local delays = {}
    for _, task in ipairs(H.scheduled) do delays[#delays + 1] = task.seconds end
    assert(#delays >= 3, "pull, highlights and the update check should each get their own step")
end)

test("an unreachable server ends the open-time work after one try", function()
    local plugin = H.newPlugin()
    H.respond = function() return nil, "timeout" end
    plugin:onReaderReady()
    runOpenSteps()
    eq(#H.requests, 1, "the other steps should not each wait on a dead network")
end)

test("test connection never waits longer than a few seconds", function()
    local plugin = H.newPlugin()
    H.respond = function() return nil, "timeout" end
    plugin:testConnection()
    for _, task in ipairs(H.scheduled) do task.fn() end
    assert(#H.timeouts >= 1)
    for _, t in ipairs(H.timeouts) do assert(t.total <= 8, "waited " .. tostring(t.total)) end
end)



test("after a server failed to answer, the next book opens do not wait on it again", function()
    local plugin = H.newPlugin()
    H.respond = function() return nil, "timeout" end
    plugin:onReaderReady(); runOpenSteps()
    local first = #H.requests
    H.scheduled = {}
    plugin:onReaderReady(); runOpenSteps()
    eq(#H.requests, first, "no request should be made while the server is known to be unreachable")
end)

-- ===== the home page =====
local function pageText(widget)
    if widget.item_table then
        local lines = {}
        for _, row in ipairs(widget.item_table) do lines[#lines + 1] = row.text .. " " .. tostring(row.mandatory or "") end
        return table.concat(lines, "\n")
    end
    return table.concat(H.widgetText(widget), "\n")
end
local function openHome(plugin, respond, plain)
    -- The tests below are about the designed page unless they ask for the plain list.
    G_reader_settings:saveSetting("mochil_designed_page", not plain)
    G_reader_settings:saveSetting("mochil_page_pending", false)
    H.respond = respond
    plugin:showHome()
    return H.shown[#H.shown]
end

test("the home page shows what is in progress and the size of the library", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", server = "https://books.test", books = 5758, series = 1191, authors = 1118, collections = 1,
        continue = {{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.05}, {title = "1984", authors = {"George Orwell"}, fraction = 0.34}}} end)
    local text = pageText(page)
    for _, expected in ipairs{"Neuromancer", "William Gibson", "5%", "1984", "34%", "5758", "1191", "1118", "Collections   1"} do
        assert(text:find(expected, 1, true), "missing '" .. expected .. "' in:\n" .. text)
    end
    assert(text:find("kevin", 1, true) and text:find("books.test", 1, true), "the account and server belong in the header:\n" .. text)
end)

test("the home page still opens when the server cannot be reached, and says why", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return nil, "host or service not found" end)
    local text = pageText(page)
    assert(text:find("Could not reach Mochil", 1, true), text)
    assert(text:find("looked up", 1, true), "the reason should be in plain words:\n" .. text)
    assert(not text:find("Wi%-Fi"))
    assert(H.findButton(page, "Try again"), "there must be a way to try again")
end)

test("a book on the shelf twice is listed once", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0,
        continue = {{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.04}, {title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.04}, {title = "1984", authors = {"George Orwell"}, fraction = 0.03}}} end)
    local _, count = pageText(page):gsub("Neuromancer", "")
    eq(count, 1)
end)

test("continue reading distinguishes same-title editions by identity and narration metadata", function()
    local plugin = H.newPlugin()
    local home = {continue = {
        {title="Neuromancer", authors={"William Gibson"}, fraction=0.1, book={id="a", library="books", format="epub"}},
        {title="Neuromancer", authors={"William Gibson"}, fraction=0.2, book={id="b", library="books", format="epub", read_along=true}},
    }}
    local rows = plugin:homeRows(home)
    local titles = {}
    for _, item in ipairs(rows) do if item.text:find("Neuromancer", 1, true) then titles[#titles + 1] = item end end
    eq(#titles, 2, "distinct editions must not be collapsed")
    assert(not titles[1].text:find("Read.along"), "ordinary EPUB is not read-along")
    assert(titles[2].text:find("Read.along"), "read-along is identified")
end)

test("nothing in progress is said plainly instead of leaving a gap", function()
    local plugin = H.newPlugin()
    local text = pageText(openHome(plugin, function() return 200, H.encode{account = "kevin", books = 0, series = 0, authors = 0, collections = 0, continue = {}} end))
    assert(text:find("Nothing in progress", 1, true), text)
end)

test("the home page only waits a few seconds for the server", function()
    local plugin = H.newPlugin()
    openHome(plugin, function() return nil, "timeout" end)
    assert(#H.timeouts >= 1)
    for _, t in ipairs(H.timeouts) do assert(t.total <= 8, "waited " .. tostring(t.total)) end
end)

test("refreshing replaces the page instead of stacking another on top", function()
    local plugin = H.newPlugin()
    local first = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    local closedBefore = #H.closed
    plugin:showHome(true)
    eq(#H.closed, closedBefore + 1, "the previous page should be closed")
    assert(H.shown[#H.shown] ~= first)
end)

test("Settings on the page opens the plugin's settings, and the page works without a book", function()
    local plugin, ui = H.newPlugin()
    ui.document = nil
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    H.findButton(page, "Settings").callback()
    local settings = H.shown[#H.shown]
    eq(settings.title, "Mochil settings")
    assert(#settings.item_table >= 8)
end)

test("KOReader overwriting the plugin's version field from _meta.lua does not confuse updates", function()
    local plugin = H.newPlugin()
    getmetatable(plugin).__index.version = "0.0.1" -- what KOReader does from the package's _meta.lua
    offer("1.0.1")
    -- the running release is newer than 1.0.1, so no update may be offered even though the old label says 0.0.1
    plugin:checkForUpdate(true)
    assert(H.shown[#H.shown].text:find("up to date", 1, true), "the label from _meta.lua was used instead of the real release: " .. H.shown[#H.shown].text)
end)

test("the page's buttons: Refresh reopens it, Files closes it", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    local closed = #H.closed
    H.findButton(page, "Refresh").callback()
    eq(#H.closed, closed + 1, "Refresh should close the old page")
    local again = H.shown[#H.shown]
    assert(again ~= page, "Refresh should open a new page")
    closed = #H.closed
    H.findButton(again, "Files").callback()
    eq(#H.closed, closed + 1, "Files should close the page and leave KOReader's file browser")
end)

test("Files leaves a book that is open for the file browser instead of showing that book again", function()
    local homed = 0
    local plugin = H.newPlugin{onHome = function() homed = homed + 1 end}
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    H.findButton(page, "Files").callback()
    eq(homed, 1, "with a book open, Files closes it and goes to the file browser")
    homed = 0
    local bare = H.newPlugin{document = false, onHome = function() homed = homed + 1 end}
    local fileManagerPage = openHome(bare, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    H.findButton(fileManagerPage, "Files").callback()
    eq(homed, 0, "already in the file browser: closing the page is enough")
end)

test("when the server cannot be reached the page offers Try again", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return nil, "timeout" end)
    local retry = H.findButton(page, "Try again")
    assert(retry, "no way to try again:\n" .. pageText(page))
    H.respond = function() return 200, H.encode{account = "kevin", books = 7, series = 1, authors = 1, collections = 0, continue = {}} end
    retry.callback()
    assert(pageText(H.shown[#H.shown]):find("Go to", 1, true), "trying again should show the page")
end)

test("designed home exposes every returned edition through Show all", function()
    local plugin = H.newPlugin()
    local entries = {}
    for i = 1, 12 do entries[i] = {title = "Edition " .. i, authors = {"A"}, fraction = 0.1, book = {id = tostring(i), library = "L", title = "Edition " .. i}} end
    local page = openHome(plugin, function(req)
        if req.url:find("/continue?", 1, true) then return 200, H.encode{total = #entries, continue = entries} end
        return 200, H.encode{continue = {entries[1]}}
    end)
    local button = H.findButton(page, "Show all")
    assert(button, "home must expose hidden entries")
    button.callback()
    local list = H.shown[#H.shown]
    assert(list.item_table, "Show all opens a paginated menu")
    local found = 0
    for _, row in ipairs(list.item_table) do if row.text:find("Edition ", 1, true) then found = found + 1; assert(row.callback) end end
    eq(found, 12)
end)

test("Show all pages preserve edition identity, formatting and callbacks", function()
    local plugin = H.newPlugin()
    local selected
    plugin.openBook = function(_, book) selected = book end
    local page = openHome(plugin, function(req)
        if req.url:find("offset=30", 1, true) then
            return 200, H.encode{total=31, continue={{title="Chapter 1", authors={"A"}, book={library="L", id="second", series="Saga", read_along=true}}}}
        elseif req.url:find("/continue?", 1, true) then
            local entries = {{title="Chapter 1", authors={"A"}, book={library="L", id="first", series="Saga"}}}
            for i=2,30 do entries[i] = {title="Book " .. i, book={library="L", id=tostring(i)}} end
            return 200, H.encode{total=31, continue=entries}
        end
        return 200, H.encode{continue={}}
    end)
    H.findButton(page, "Show all").callback()
    local list = H.shown[#H.shown]
    assert(pageText(list):find("Loaded 30 / 31", 1, true))
    H.findButton(list, "Show more").callback()
    assert(fired("offset=30"))
    assert(pageText(list):find("Loaded 31 / 31", 1, true))
    local editions = {}
    for _, row in ipairs(list.item_table) do if row.text:find("Saga — Chapter 1", 1, true) then editions[#editions+1] = row end end
    eq(#editions, 2)
    assert(not editions[1].text:find("Read-along", 1, true))
    assert(editions[2].text:find("Read-along", 1, true))
    editions[1].callback(); eq(selected.id, "first")
    editions[2].callback(); eq(selected.id, "second")
end)

test("Show all falls back to home entries on unsupported or failed requests and retries", function()
    for _, failure in ipairs{404, 500, "network", "invalid"} do
        H.reset()
        local plugin = H.newPlugin()
        local selected
        plugin.openBook = function(_, book) selected = book.id end
        local fail = true
        local page = openHome(plugin, function(req)
            if req.url:find("/continue?", 1, true) then
                if not fail then return 200, H.encode{total=1, continue={{title="Remote", book={id="remote", library="L"}}}} end
                if failure == "network" then return nil, "timeout" end
                if failure == "invalid" then return 200, H.encode{total=3} end
                return failure, ""
            end
            return 200, H.encode{continue={{title="Cached", book={id="cached", library="L"}}}}
        end)
        H.findButton(page, "Show all").callback()
        local list = H.shown[#H.shown]
        assert(pageText(list):find("Could not load the full list", 1, true))
        local cached = H.findButton(list, "Cached")
        assert(cached); cached.callback(); eq(selected, "cached")
        assert(not pageText(list):find("Loaded 1 / 1", 1, true))
        fail = false; H.findButton(list, "Try again").callback()
        assert(pageText(list):find("Remote", 1, true))
        assert(not pageText(list):find("Cached", 1, true))
    end
end)

test("failed full list with no cached entries does not claim nothing is in progress", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function(req)
        if req.url:find("/continue?", 1, true) then return nil, "timeout" end
        return 200, H.encode{continue={}}
    end)
    H.findButton(page, "Show all").callback()
    local text = pageText(H.shown[#H.shown])
    assert(not text:find("Nothing in progress yet", 1, true), text)
    assert(text:find("No cached home entries", 1, true), text)
end)

test("Show all retains fetched pages when a later request fails", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function(req)
        if req.url:find("offset=30", 1, true) then return nil, "timeout" end
        if req.url:find("/continue?", 1, true) then
            local entries = {}; for i=1,30 do entries[i]={title="Book " .. i} end
            return 200, H.encode{total=31, continue=entries}
        end
        return 200, H.encode{continue={{title="Home cached"}}}
    end)
    H.findButton(page, "Show all").callback()
    local list = H.shown[#H.shown]
    H.findButton(list, "Show more").callback()
    assert(pageText(list):find("Could not load the full list", 1, true))
    assert(pageText(list):find("Book 30", 1, true))
    assert(not pageText(list):find("Home cached", 1, true))
    assert(H.findButton(list, "Try again"))
end)

test("continue list keeps long titles and wraps rows", function()
    local plugin = H.newPlugin()
    local long = string.rep("Long title ", 20)
    local list = openHome(plugin, function() return 200, H.encode{continue={{title=long, authors={}}}} end, true)
    eq(list.single_line, false)
    assert(pageText(list):find(long, 1, true))
end)

test("designed home toolbar uses native icons and preserves actions", function()
    local plugin = H.newPlugin()
    local calls = {}
    plugin.browseKind = function(_, kind) calls[#calls + 1] = kind end
    plugin.showSettingsPage = function() calls[#calls + 1] = "settings" end
    plugin.showFiles = function() calls[#calls + 1] = "files" end
    local page = openHome(plugin, function() return 200, H.encode{continue={}} end)
    local function findIcon(node, icon, seen)
        if type(node) ~= "table" then return end
        seen = seen or {}; if seen[node] then return end; seen[node] = true
        if node.icon == icon and node.callback then return node end
        for key, child in pairs(node) do
            if key ~= "show_parent" then local found = findIcon(child, icon, seen); if found then return found end end
        end
    end
    for _, item in ipairs{{"appbar.search", "books"}, {"appbar.settings", "settings"}, {"appbar.filebrowser", "files"}} do
        local button = assert(findIcon(page, item[1]), "missing icon " .. item[1])
        assert(button.hold_callback, "icon needs an explanatory hold action")
        button.hold_callback()
        assert(H.shown[#H.shown].text and #H.shown[#H.shown].text > 0, "hold must explain the icon")
        button.callback(); eq(calls[#calls], item[2])
    end
    assert(findIcon(page, "cre.render.reload"), "missing refresh icon")
end)

local function findAllIn(widget, predicate, found, seen)
    found, seen = found or {}, seen or {}
    if type(widget) ~= "table" or seen[widget] then return found end
    seen[widget] = true
    if predicate(widget) then found[#found + 1] = widget end
    for key, child in pairs(widget) do if key ~= "show_parent" then findAllIn(child, predicate, found, seen) end end
    return found
end

test("designed home separates sections with a visible divider under each heading", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end)
    local dividers = findAllIn(page, function(w) return w.is_divider == true end)
    assert(#dividers >= 1, "the 'Go to' heading should be followed by a divider line")
end)

test("designed home keeps title as the strong element without coloring the author gray", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{continue = {{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3}}} end)
    local titleNode = findAllIn(page, function(w) return w.text == "Neuromancer" end)[1]
    local authorNode = findAllIn(page, function(w) return w.text == "William Gibson" end)[1]
    assert(titleNode, "title text not found")
    assert(authorNode, "author text not found")
    assert(titleNode.bold == true, "the title stays the strong element")
    assert(not authorNode.bold, "the author line is not bold, so hierarchy still reads clearly")
    assert(titleNode.face and authorNode.face and titleNode.face.size ~= authorNode.face.size,
        "title and author must differ in size since color can no longer carry hierarchy")
end)

test("no card text uses a gray/dithered color; hierarchy comes from weight and size alone", function()
    local plugin = H.newPlugin()
    local Blitbuffer = require("ffi/blitbuffer")
    local page = openHome(plugin, function() return 200, H.encode{continue = {
        {title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3},
    }} end)
    local grays = {Blitbuffer.COLOR_GRAY, Blitbuffer.COLOR_DARK_GRAY, Blitbuffer.COLOR_LIGHT_GRAY}
    local texts = findAllIn(page, function(w) return type(w.text) == "string" and #w.text > 0 end)
    assert(#texts >= 3, "expected at least the title, author and percentage text widgets")
    for _, node in ipairs(texts) do
        for _, gray in ipairs(grays) do
            assert(node.fgcolor ~= gray, "card text must not use a gray/dithered color: " .. tostring(node.text))
        end
    end
end)

test("the title row holds both the title and the percentage on one line", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{continue = {{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3}}} end)
    local row = findAllIn(page, function(w) return w.is_title_row == true end)[1]
    assert(row, "no title row found")
    local hasTitle, hasPercent, titleIndex, percentIndex = false, false, nil, nil
    for index, child in ipairs(row) do
        if type(child) == "table" then
            if child.text == "Neuromancer" then hasTitle = true; titleIndex = index end
            if child.text == "30%" then hasPercent = true; percentIndex = index end
        end
    end
    assert(hasTitle, "title row must contain the title text")
    assert(hasPercent, "title row must contain the inline percentage text")
    assert(titleIndex < percentIndex, "the title must sit left of the percentage")
end)

test("the percentage stays flush at a fixed right edge and never wraps, regardless of title length", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{continue = {
        {title = "Hi", authors = {}, fraction = 0.3},
        {title = "A Rather Much Longer Book Title Than The First One", authors = {}, fraction = 0.3},
    }} end)
    local rows = findAllIn(page, function(w) return w.is_title_row == true end)
    eq(#rows, 2, "expected one title row per card")
    local totals = {}
    for i, row in ipairs(rows) do
        local sum, percentNode = 0, nil
        for _, child in ipairs(row) do
            if type(child) == "table" then
                if child.getSize then sum = sum + child:getSize().w
                elseif child.width then sum = sum + child.width end
                if child.text == "30%" then percentNode = child end
            end
        end
        totals[i] = sum
        assert(percentNode, "percentage text missing from title row " .. i)
        assert(percentNode.max_width and percentNode.max_width > 0, "percentage must have a fixed width so it cannot wrap")
        assert(percentNode.truncate_with_ellipsis == false, "the short percentage label must not be truncated/wrapped")
    end
    assert(math.abs(totals[1] - totals[2]) < 1,
        "the title row's total width (title + spacer + percentage) must be constant regardless of title length, " ..
        "proving the percentage is pinned to a fixed right edge: " .. totals[1] .. " vs " .. totals[2])
end)

test("the continue reading progress indicator is a thin bar with a light track, not a thick bordered widget", function()
    local plugin = H.newPlugin()
    local Blitbuffer = require("ffi/blitbuffer")
    local page = openHome(plugin, function() return 200, H.encode{continue = {{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3}}} end)
    local bar = findAllIn(page, function(w) return w.percentage ~= nil end)[1]
    assert(bar, "no progress indicator found")
    assert(bar.height ~= nil and bar.height <= 4, "the indicator must be a thin bar (height <= 4), not the old thicker style: height=" .. tostring(bar.height))
    eq(bar.bordersize, 0, "a thin minimal indicator must not use a thick border")
    eq(bar.bgcolor, Blitbuffer.COLOR_LIGHT_GRAY, "the unfilled track should read as a light background")
    -- The bar must not itself be in its own row with the percentage any more (that moved to the title row).
    local row = findAllIn(page, function(w) return w.is_progress_row == true end)[1]
    assert(not row, "the progress bar must no longer be merged into a separate percentage row")
end)

test("every continue reading card shares the same corner radius and padding", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{continue = {
        {title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3},
        {title = "1984", authors = {"George Orwell"}, fraction = 0.6},
    }} end)
    local cards = findAllIn(page, function(w) return w.radius ~= nil and w.bordersize ~= nil end)
    assert(#cards >= 2, "expected at least two styled cards")
    local radius, padding = cards[1].radius, cards[1].padding
    for _, card in ipairs(cards) do
        eq(card.radius, radius, "card corner radius must be consistent")
        eq(card.padding, padding, "card padding must be consistent")
    end
    assert(radius > 0 and radius <= 8, "corner radius should stay tight/minimal, not a rounded pill: got " .. radius)
    assert(padding >= 10, "internal padding should read as generous breathing room, not cramped: got " .. padding)
end)

test("continue reading cards get an explicit fixed height regardless of author line length", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{continue = {
        {title = "Solo Author Book", authors = {"One Author"}, fraction = 0.2},
        {title = "Multi Author Book", authors = {"KARASUYAMA Eiji", "KISHI Yuusuke"}, fraction = 0.4},
    }} end)
    local cards = findAllIn(page, function(w) return w.radius ~= nil and w.bordersize ~= nil end)
    eq(#cards, 2, "expected two continue reading cards")
    assert(cards[1].height ~= nil and cards[1].height > 0, "cards must declare an explicit fixed height, not rely on auto-sizing from their content")
    eq(cards[1].height, cards[2].height, "a multi-author card must be exactly as tall as a single-author card")
end)

test("long title and author text is truncated with an ellipsis, but the tap target keeps the full identity", function()
    local plugin = H.newPlugin()
    local longAuthors = {}
    for i = 1, 12 do longAuthors[i] = "Author Name Number " .. i end
    local authorText = table.concat(longAuthors, ", ")
    local longTitle = string.rep("Very Long Title Segment ", 10):match("^%s*(.-)%s*$")
    local detail = {id = "full-id", library = "books", format = "epub", title = longTitle}
    local page = openHome(plugin, function() return 200, H.encode{continue = {
        {title = longTitle, authors = longAuthors, fraction = 0.1, book = detail},
    }} end)
    local titleNode = findAllIn(page, function(w) return type(w.text) == "string" and longTitle:find(w.text:gsub("…$", ""), 1, true) == 1 end)[1]
    assert(titleNode, "title widget not found")
    assert(titleNode.text ~= longTitle, "the on-screen title must actually be shortened: " .. titleNode.text)
    eq(titleNode.text:sub(-3), "…", "a long title must be truncated with a trailing ellipsis: " .. titleNode.text)
    local authorNode = findAllIn(page, function(w) return type(w.text) == "string" and authorText:find(w.text:gsub("…$", ""), 1, true) == 1 end)[1]
    assert(authorNode, "author widget not found")
    assert(authorNode.text ~= authorText, "the on-screen author line must actually be shortened: " .. authorNode.text)
    eq(authorNode.text:sub(-3), "…", "a long multi-author line must be truncated with a trailing ellipsis: " .. authorNode.text)
    local card = findAllIn(page, function(w) return type(w.onTapCard) == "function" end)[1]
    assert(card, "no tappable card found")
    local opened
    plugin.openBook = function(_, book) opened = book end
    card.onTapCard()
    eq(opened.id, "full-id", "tapping must still navigate using the untruncated book identity")
    eq(opened.title, longTitle, "the full original title must reach the open callback, not the truncated label")
end)

test("plain home offers book search", function()
    for _, plain in ipairs{true} do
        local plugin = H.newPlugin()
        local kind
        plugin.browseKind = function(_, value) kind = value end
        local page = openHome(plugin, function() return 200, H.encode{continue={}} end, plain)
        local search = H.findButton(page, "Search books")
        assert(search, "missing book search")
        search.callback()
        eq(kind, "books")
    end
end)

test("small portrait and landscape screens use the paginated home without changing the preference", function()
    local screen = require("device").screen
    local oldWidth, oldHeight = screen.getWidth, screen.getHeight
    for _, size in ipairs{{400, 550}, {800, 400}} do
        screen.getWidth = function() return size[1] end
        screen.getHeight = function() return size[2] end
        local page = openHome(H.newPlugin(), function() return 200, H.encode{continue = {{title="Visible", authors={}}}} end)
        assert(page.item_table, "small screen must use paginated list")
        assert(pageText(page):find("Visible", 1, true))
        eq(G_reader_settings:readSetting("mochil_designed_page"), true)
    end
    screen.getWidth, screen.getHeight = oldWidth, oldHeight
end)

test("designed home falls back when its fixed controls do not fit", function()
    local saved = package.preload["ui/widget/titlebar"]
    package.loaded["ui/widget/titlebar"] = nil
    package.preload["ui/widget/titlebar"] = function() return {new = function(_, fields) fields.getSize = function() return {w=600, h=900} end; return fields end} end
    local page = openHome(H.newPlugin(), function() return 200, H.encode{continue={{title="Visible", authors={}}}} end)
    package.preload["ui/widget/titlebar"] = saved
    package.loaded["ui/widget/titlebar"] = nil
    assert(page.item_table and pageText(page):find("Visible", 1, true), "overflow must use the list")
end)

test("only as many book cards as fit on the screen are shown", function()
    local plugin = H.newPlugin()
    local many = {}
    for index = 1, 30 do many[index] = {title = "Book " .. index, authors = {"Author"}, fraction = 0.1} end
    local text = pageText(openHome(plugin, function() return 200, H.encode{account = "kevin", books = 30, series = 1, authors = 1, collections = 0, continue = many} end))
    local _, shown = text:gsub("Book %d+", "")
    assert(shown >= 1 and shown < 30, "expected the cards to be limited to the screen, got " .. shown)
end)

test("if the designed layout cannot be built, the plain list is shown instead of nothing", function()
    local plugin = H.newPlugin()
    package.preload["ui/widget/titlebar"] = function() error("this KOReader has no such widget") end
    package.loaded["ui/widget/titlebar"] = nil
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end)
    package.preload["ui/widget/titlebar"] = H.widgetModules["ui/widget/titlebar"]
    package.loaded["ui/widget/titlebar"] = nil
    assert(page.item_table, "the fallback list should be on screen")
    assert(pageText(page):find("Books", 1, true))
end)

test("a layout that fails when measured falls back to the list instead of being shown", function()
    local plugin = H.newPlugin()
    local real = H.widgetModules["ui/widget/container/framecontainer"]
    package.preload["ui/widget/container/framecontainer"] = function()
        return {new = function(_, fields) fields.getSize = function() error("cannot lay this out") end; return fields end}
    end
    package.loaded["ui/widget/container/framecontainer"] = nil
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end)
    package.preload["ui/widget/container/framecontainer"] = real
    package.loaded["ui/widget/container/framecontainer"] = nil
    assert(page.item_table, "the broken layout must not be shown")
end)

test("the plain list is the default; the designed page has to be switched on", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_designed_page", nil)
    H.respond = function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end
    plugin:showHome()
    assert(H.shown[#H.shown].item_table, "the default page should be the plain list")
end)

test("the designed page is marked pending while it is drawn and cleared once it has been on screen", function()
    local plugin = H.newPlugin()
    openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end)
    eq(G_reader_settings:readSetting("mochil_page_pending"), true, "it must be marked before it is drawn")
    local cleared = false
    for _, task in ipairs(H.scheduled) do if task.seconds == 3 then task.fn(); cleared = true end end
    assert(cleared, "nothing clears the mark once the page is up")
    eq(G_reader_settings:readSetting("mochil_page_pending"), false)
end)

test("an old refresh timer cannot clear the new page's pending marker", function()
    local plugin = H.newPlugin()
    openHome(plugin, function() return 200, H.encode{books=1, continue={}} end)
    local old
    for _, task in ipairs(H.scheduled) do if task.seconds == 3 then old = task.fn end end
    plugin:showHome(true)
    old()
    eq(G_reader_settings:readSetting("mochil_page_pending"), true, "the new draw still needs protection")
end)

test("closing the designed page before its timer does not trigger recovery on restart", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{books=1, continue={}} end)
    page:onCloseWidget()
    eq(G_reader_settings:readSetting("mochil_page_pending"), false, "a clean close clears the marker")
    local restarted = H.newPlugin()
    restarted:showHome()
    assert(not H.shown[#H.shown].item_table, "a clean close keeps the designed page")
end)

test("an interrupted page falls back without erasing the preference, and retries after restart", function()
    G_reader_settings:saveSetting("mochil_designed_page", true)
    G_reader_settings:saveSetting("mochil_page_pending", true)
    local plugin = H.newPlugin(); plugin:init()
    eq(G_reader_settings:readSetting("mochil_designed_page"), true, "recovery must not overwrite user choice")
    eq(G_reader_settings:readSetting("mochil_designed_page_crashed"), true)
    H.respond = function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end
    plugin:showHome()
    assert(H.shown[#H.shown].item_table, "the recovery instance uses the plain list")
    local restarted = H.newPlugin(); restarted:init(); restarted:showHome()
    assert(not H.shown[#H.shown].item_table, "a later restart retries the chosen design")
end)

test("changing the home preference flushes it for restart", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_designed_page", false)
    local flushed
    G_reader_settings.flush = function() flushed = G_reader_settings:readSetting("mochil_designed_page") end
    for _, entry in ipairs(plugin:menuEntries()) do
        if entry.text and entry.text:find("Designed home page", 1, true) then entry.callback(); break end
    end
    G_reader_settings.flush = nil
    eq(flushed, true, "switch should persist immediately")
end)

test("the plugin settings offer the designed page as a switch", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_designed_page", false)
    local entry
    for _, candidate in ipairs(plugin:menuEntries()) do if candidate.text and candidate.text:find("Designed home page", 1, true) then entry = candidate end end
    assert(entry, "no switch for the designed page")
    eq(entry.checked_func(), false)
    entry.callback()
    eq(entry.checked_func(), true)
end)

-- ===== tapping rows on the list pages =====
local function rowNamed(page, text)
    for _, row in ipairs(page.item_table) do
        local label = row.text or (row.text_func and row.text_func())
        if label == text or (label and label:find(text, 1, true) == 1) then return row end
    end
end

test("tapping a row on the home list runs it and keeps the page open", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end, true)
    local closed = #H.closed
    page:onMenuSelect(rowNamed(page, "Plugin version"))
    eq(#H.closed, closed, "tapping a row must not close the page")
    assert(H.shown[#H.shown].text and H.shown[#H.shown].text:find("Running version", 1, true), "the row's own action should have run")
    assert(page.updates and page.updates >= 1, "the rows should be refreshed after a tap")
end)

test("the close button still closes the page", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 1, series = 1, authors = 1, collections = 0, continue = {}} end, true)
    local closed = #H.closed
    page.close_callback()
    eq(#H.closed, closed + 1)
end)

test("a switch on the settings page flips, shows On or Off, and the page stays open", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_designed_page", false)
    plugin:showSettingsPage()
    local page = H.shown[#H.shown]
    local row
    for _, candidate in ipairs(page.item_table) do if candidate.text and candidate.text:find("Designed home page", 1, true) then row = candidate end end
    assert(row and row.mandatory_func, "the switch should show its state")
    eq(row.mandatory_func(), "Off")
    local closed = #H.closed
    page:onMenuSelect(row)
    eq(#H.closed, closed, "flipping a switch must not close the page")
    eq(row.mandatory_func(), "On")
    eq(G_reader_settings:readSetting("mochil_designed_page"), true)
end)

test("a group of choices opens inside the page and marks the one in use", function()
    local plugin = H.newPlugin()
    plugin:setSyncPolicy("manual")
    plugin:showSettingsPage()
    local page = H.shown[#H.shown]
    local group = rowNamed(page, "Automatic push")
    assert(group and group.sub_item_table, "the choices should be a sub-list")
    page:onMenuSelect(group)
    eq(page.switched, true, "the choices should open in the same page")
    local marked
    for _, choice in ipairs(group.sub_item_table) do if choice.mandatory_func and choice.mandatory_func() == "✓" then marked = choice.text end end
    eq(marked, "Manual only")
end)

local function pushes()
    local n = 0
    for _, request in ipairs(H.requests) do if request.method == "PUT" or request.method == "POST" then n = n + 1 end end
    return n
end
local function runScheduled()
    local tasks = {table.unpack(H.scheduled)}
    H.scheduled = {}
    for _, task in ipairs(tasks) do task.fn() end
end

test("automatic push offers four choices and every stored value is read as itself", function()
    for _, stored in ipairs({"close", "chapter", "page", "manual"}) do
        G_reader_settings:saveSetting("mochil_sync_policy", stored)
        local plugin = H.newPlugin()
        eq(plugin:syncPolicy(), stored)
        plugin:showSettingsPage()
        local group = rowNamed(H.shown[#H.shown], "Automatic push")
        eq(#group.sub_item_table, 4)
        eq(group.sub_item_table[1].text, "When closing"); eq(group.sub_item_table[2].text, "After changing chapter")
        eq(group.sub_item_table[3].text, "After every page"); eq(group.sub_item_table[4].text, "Manual only")
        local checked = 0
        for _, choice in ipairs(group.sub_item_table) do if choice.checked_func() then checked = checked + 1 end end
        eq(checked, 1)
        eq(G_reader_settings:readSetting("mochil_sync_policy"), stored)
        local legacy = plugin:menuEntries()
        local sub
        for _, entry in ipairs(legacy) do if entry.text == "Automatic push" then sub = entry.sub_item_table end end
        eq(#sub, 4, "the legacy menu offers the same four")
        eq(sub[2].text, "After changing chapter"); eq(sub[3].text, "After every page")
    end
    G_reader_settings:saveSetting("mochil_sync_policy", "page")
    eq(H.newPlugin():showSettingsPage(), nil)
    eq(rowNamed(H.shown[#H.shown], "Automatic push").mandatory_func(), "After every page")
    G_reader_settings:saveSetting("mochil_sync_policy", "nonsense")
    eq(H.newPlugin():syncPolicy(), "close", "unknown values fall back to close")
    G_reader_settings:saveSetting("mochil_sync_policy", nil)
    eq(H.newPlugin():syncPolicy(), "close")
    local plugin = H.newPlugin()
    plugin:setSyncPolicy("page")
    eq(G_reader_settings:readSetting("mochil_sync_policy"), "page")
    plugin:setSyncPolicy("chapter")
    eq(G_reader_settings:readSetting("mochil_sync_policy"), "chapter")
    G_reader_settings:saveSetting("mochil_sync_policy", nil)
end)

test("every page: a page turn pushes once after a pause, only when online and not manual", function()
    G_reader_settings:saveSetting("mochil_sync_policy", "page")
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{ok = true} end
    H.requests, H.scheduled = {}, {}
    plugin:onPageUpdate(); plugin:onPosUpdate(); plugin:onPageUpdate()
    eq(#H.scheduled, 1, "turns inside the pause collapse into one scheduled push")
    eq(H.scheduled[1].seconds, 2)
    eq(pushes(), 0, "nothing is sent while the pages are still turning")
    runScheduled()
    eq(pushes(), 1)
    -- offline: the callback does not send
    H.requests = {}
    package.loaded["ui/network/manager"].online = false
    plugin:onPageUpdate(); runScheduled()
    eq(pushes(), 0, "offline pushes nothing")
    package.loaded["ui/network/manager"].online = true
    -- manual: nothing is even scheduled
    plugin:setSyncPolicy("manual")
    H.scheduled = {}
    plugin:onPageUpdate(); plugin:onPosUpdate()
    eq(#H.scheduled, 0, "manual schedules nothing mid-read")
    runScheduled()
    eq(pushes(), 0)
    G_reader_settings:saveSetting("mochil_sync_policy", nil)
end)

test("every chapter: only crossing into another chapter pushes", function()
    G_reader_settings:saveSetting("mochil_sync_policy", "chapter")
    local fragment = 1
    local plugin = H.newPlugin()
    plugin.ui.document.getXPointer = function() return "/body/DocFragment[" .. fragment .. "]/p[1]" end
    H.respond = function() return 200, H.encode{ok = true} end
    plugin:onReaderReady()
    H.requests, H.scheduled = {}, {}
    plugin:onPageUpdate()
    eq(#H.scheduled, 0, "same chapter: nothing scheduled")
    fragment = 2
    plugin:onPageUpdate(); plugin:onPosUpdate()
    eq(#H.scheduled, 1, "a new chapter schedules one debounced push")
    eq(H.scheduled[1].seconds, 2)
    eq(pushes(), 0)
    runScheduled()
    eq(pushes(), 1)
    plugin:onPageUpdate()
    eq(#H.scheduled, 0, "still the same chapter")
    plugin:setSyncPolicy("manual")
    fragment = 3
    plugin:onPageUpdate()
    eq(#H.scheduled, 0, "manual schedules nothing")
    -- fixed-layout books have no fragments: no chapter push, no error
    plugin:setSyncPolicy("chapter")
    plugin.ui.document.info.has_pages = true
    plugin:onPageUpdate()
    eq(#H.scheduled, 0)
    G_reader_settings:saveSetting("mochil_sync_policy", nil)
end)

test("closing the book schedules nothing mid-read and still pushes the final position", function()
    for _, stored in ipairs({"close", "chapter", "page"}) do
        G_reader_settings:saveSetting("mochil_sync_policy", stored)
        local plugin = H.newPlugin()
        H.respond = function() return 200, H.encode{ok = true} end
        H.requests, H.scheduled = {}, {}
        if stored == "close" then plugin:onPageUpdate(); eq(#H.scheduled, 0, "close policy: page turns schedule nothing") end
        plugin:onCloseDocument()
        assert(pushes() >= 1, stored .. ": the close must push")
        plugin = H.newPlugin()
        H.requests = {}
        plugin:onSuspend()
        assert(pushes() >= 1, stored .. ": suspend must push")
    end
    G_reader_settings:saveSetting("mochil_sync_policy", "manual")
    local plugin = H.newPlugin()
    H.requests = {}
    plugin:onCloseDocument(); plugin:onSuspend()
    eq(pushes(), 0)
    G_reader_settings:saveSetting("mochil_sync_policy", nil)
end)

test("the settings page keeps only real settings and still has the actions that have no other way in", function()
    local plugin = H.newPlugin()
    plugin:showSettingsPage()
    local page = H.shown[#H.shown]
    eq(rowNamed(page, "Search all libraries"), nil)
    eq(rowNamed(page, "Check for updates automatically"), nil)
    assert(rowNamed(page, "Check for updates"), "manual check stays")
    assert(rowNamed(page, "Test connection"), "test connection moved to the Plugin section")
    -- Not reachable from the home page, so they stay.
    assert(rowNamed(page, "Reading statistics"))
    assert(rowNamed(page, "On hold and dropped"))
    assert(rowNamed(page, "Download queue"))
    local legacy = {}
    for _, entry in ipairs(plugin:menuEntries()) do legacy[#legacy + 1] = entry.text or "" end
    local joined = table.concat(legacy, "|")
    eq(joined:find("automatically", 1, true), nil)
    assert(joined:find("Test connection", 1, true))
end)

test("a row that is switched off does nothing when tapped", function()
    local plugin = H.newPlugin()
    plugin:showSettingsPage()
    local page = H.shown[#H.shown]
    local server = rowNamed(page, "Server: ")
    assert(server and server.select_enabled_func and not server.select_enabled_func())
    local closed, shown = #H.closed, #H.shown
    page:onMenuSelect(server)
    eq(#H.closed, closed); eq(#H.shown, shown)
end)

test("a page that cannot be drawn is caught before it is shown, and the reason is kept", function()
    local plugin = H.newPlugin()
    local real = H.widgetModules["ui/widget/container/inputcontainer"]
    package.preload["ui/widget/container/inputcontainer"] = function()
        return {new = function(_, fields) fields.paintTo = function() error("attempt to index a nil value (drawing)") end; return fields end}
    end
    package.loaded["ui/widget/container/inputcontainer"] = nil
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end)
    package.preload["ui/widget/container/inputcontainer"] = real
    package.loaded["ui/widget/container/inputcontainer"] = nil
    assert(page.item_table, "an undrawable page must not be shown")
    local reason = G_reader_settings:readSetting("mochil_designed_page_error")
    assert(reason and reason:find("drawing", 1, true), "the reason should be kept for the install details")
    plugin:showInstallDetails()
    assert(H.shown[#H.shown].text:find("could not be drawn", 1, true) and H.shown[#H.shown].text:find("drawing", 1, true))
end)

test("the page keeps a margin on the left of the cards and buttons", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {{title = "Book", authors = {"A"}, fraction = 0.1}}} end)
    local function find(widget, predicate, seen)
        seen = seen or {}
        if type(widget) ~= "table" or seen[widget] then return nil end
        seen[widget] = true
        if predicate(widget) then return widget end
        for key, child in pairs(widget) do if key ~= "show_parent" then local found = find(child, predicate, seen); if found then return found end end end
    end
    local indent = find(page, function(w) return w.padding_left and w.padding_left > 0 end)
    assert(indent, "the content under the title bar needs a left margin")
    assert(find(indent, function(w) return w.icon == "cre.render.reload" end), "the buttons must be inside the indented area")
    assert(find(indent, function(w) return w.text == "Book" end), "the cards must be inside the indented area")
    assert(not find(indent, function(w) return w.subtitle ~= nil end), "the title bar spans the screen, outside the margin")
end)

test("the designed page has no version footer (it is under Plugin version)", function()
    local plugin = H.newPlugin()
    local text = pageText(openHome(plugin, function() return 200, H.encode{account = "kevin", books = 5, series = 1, authors = 1, collections = 0, continue = {}} end))
    assert(not text:find("Mochil " .. plugin.release, 1, true), "the footer is back:\n" .. text)
end)

-- ===== browsing the library and downloading =====
local function lastRequestFor(fragment)
    for index = #H.requests, 1, -1 do if H.requests[index].url:find(fragment, 1, true) then return H.requests[index] end end
end
local function row(page, text)
    for _, item in ipairs(page.item_table) do if item.text:find(text, 1, true) == 1 then return item end end
    error("no row '" .. text .. "' in:\n" .. pageText(page))
end
local dune = {id = "a1", library = "books", title = "Dune", authors = {"Frank Herbert"}, series = "Dune", series_index = 1, format = "epub", size = 2097152}
local neuromancer = {id = "n1", library = "books", title = "Neuromancer", authors = {"William Gibson"}, format = "epub", size = 1048576}
local function browseServer(responses)
    return function(req)
        for fragment, body in pairs(responses) do
            if req.url:find(fragment, 1, true) then return 200, H.encode(body) end
        end
        if req.url:find("/home", 1, true) then return 200, H.encode{account = "kevin", books = 2, series = 1, authors = 2, collections = 1, continue = {}} end
        return 404, "{}"
    end
end
local function downloadFolder()
    local folder = H.tmpdir()
    G_reader_settings:saveSetting("mochil_download_dir", folder)
    G_reader_settings:saveSetting("mochil_download_layout", nil)
    return folder
end

test("the Library, Authors, Series and Collections entries open lists (plain page and designed page)", function()
    for _, plain in ipairs{true, false} do
        local plugin = H.newPlugin()
        local page = openHome(plugin, browseServer{["kind=books"] = {kind = "books", total = 2, books = {dune, neuromancer}}}, plain)
        local open
        if plain then open = row(page, "Books").callback
        else
            local tile = H.findButton(page, "Library   2")
            assert(tile and tile.enabled ~= false, "the Library tile must be tappable")
            open = tile.callback
        end
        open()
        local list = H.shown[#H.shown]
        assert(list.is_menu, "tapping should open a list")
        local text = pageText(list)
        assert(text:find("Dune", 1, true) and text:find("Neuromancer", 1, true), text)
        assert(lastRequestFor("browse?kind=books&limit=30&offset=0"), "the list asks the server for a screenful")
        assert(lastRequestFor("browse").headers.Authorization:find("Bearer", 1, true), "browsing uses the device token")
    end
end)

test("a series opens to its books in volume order, and a book can be opened for download", function()
    local plugin = H.newPlugin()
    H.respond = browseServer{
        ["kind=series&limit=30&offset=0"] = {kind = "series", total = 1, groups = {{name = "Dune", library = "books", count = 2}}},
        ["series=Dune"] = {kind = "books", total = 2, books = {dune, {id = "a2", library = "books", title = "Dune Messiah", authors = {"Frank Herbert"}, series = "Dune", series_index = 2, format = "epub"}}},
    }
    plugin:showBrowse{kind = "series", title = "Series"}
    local list = H.shown[#H.shown]
    assert(row(list, "Dune").mandatory == "2", "a series shows how many books it has")
    row(list, "Dune").callback()
    local books = H.shown[#H.shown]
    assert(lastRequestFor("library=books") and lastRequestFor("series=Dune"), "the series and its library are sent to the server")
    local text = pageText(books)
    assert(text:find("1. Dune", 1, true) and text:find("2. Dune Messiah", 1, true), "volumes are numbered:\n" .. text)
    row(books, "1. Dune").callback()
    local dialog = H.shown[#H.shown]
    assert(H.findButton(dialog.buttons, "Download"), "an undownloaded book offers Download")
end)

test("a long list loads a screenful at a time", function()
    local plugin = H.newPlugin()
    local first, second = {}, {}
    for index = 1, 30 do first[index] = {id = "f" .. index, library = "books", title = "First " .. index, authors = {}, format = "epub"} end
    for index = 1, 15 do second[index] = {id = "s" .. index, library = "books", title = "Second " .. index, authors = {}, format = "epub"} end
    H.respond = function(req)
        if req.url:find("offset=30", 1, true) then return 200, H.encode{kind = "books", total = 45, books = second} end
        return 200, H.encode{kind = "books", total = 45, books = first}
    end
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    assert(row(list, "Show more").mandatory == "15", "it says how many are left")
    row(list, "Show more").callback()
    assert(lastRequestFor("offset=30"), "the next screenful starts where the last one ended")
    local text = pageText(list)
    assert(text:find("First 1 ", 1, true) and text:find("Second 15", 1, true) and not text:find("Show more", 1, true), text)
end)

test("searching a list asks the server and can be cleared", function()
    local plugin = H.newPlugin()
    H.respond = function(req)
        if req.url:find("q=gibson", 1, true) then return 200, H.encode{kind = "books", total = 1, books = {neuromancer}} end
        return 200, H.encode{kind = "books", total = 2, books = {dune, neuromancer}}
    end
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    row(list, "Search").callback()
    local dialog = H.shown[#H.shown]
    dialog.input_text = "gibson"
    H.findButton(dialog.buttons, "Search").callback()
    assert(lastRequestFor("q=gibson"), "the words are sent to the server")
    local text = pageText(list)
    assert(text:find("Neuromancer", 1, true) and not text:find("Dune", 1, true) and text:find("Search: gibson", 1, true), text)
    row(list, "Search: gibson").callback()
    assert(pageText(list):find("Dune", 1, true), "clearing the search brings the whole list back")
end)

test("a list that cannot load says why and offers to try again", function()
    local plugin = H.newPlugin()
    H.respond = function() return nil, "host or service not found" end
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    local text = pageText(list)
    assert(text:find("looked up", 1, true) and text:find("Try again", 1, true), text)
    H.respond = browseServer{["kind=books"] = {kind = "books", total = 1, books = {dune}}}
    row(list, "Try again").callback()
    assert(pageText(list):find("Dune", 1, true), "trying again loads the list")
end)

-- ===== streaming a comic page-by-page (OPDS-PSE style, no local file) =====
local streamableComic = {id = "comic", library = "manga", title = "Comic", authors = {"Artist"}, format = "cbz", streamable = true, page_count = 5}

-- Marking a book finished writes the answer back onto the book table the dialog was built from, so the shared
-- fixtures have to start each test unmarked or the next dialog offers the opposite toggle.
local fixtures = {dune, neuromancer, streamableComic}
H.reset_fixtures = function() for _, book in ipairs(fixtures) do book.finished = nil end end

test("the book dialog offers Stream only when the server marked the book streamable", function()
    local plugin = H.newPlugin()
    plugin:showBook(streamableComic)
    local dialog = H.shown[#H.shown]
    assert(H.findButton(dialog.buttons, "Stream"), "a streamable comic must offer Stream")
    assert(H.findButton(dialog.buttons, "Download"), "Download must still be offered alongside Stream")

    local notStreamable = {id = "n1", library = "books", title = "Neuromancer", authors = {"William Gibson"}, format = "epub"}
    plugin:showBook(notStreamable)
    local plain = H.shown[#H.shown]
    assert(not H.findButton(plain.buttons, "Stream"), "a non-streamable book must not offer Stream")
    assert(H.findButton(plain.buttons, "Download"), "Download is unaffected")
end)

test("the book dialog offers a mark-finished toggle that reflects the server's current status", function()
    local plugin = H.newPlugin()
    plugin:showBook(dune)
    local unfinished = H.shown[#H.shown]
    assert(H.findButton(unfinished.buttons, "Mark as finished"), "an unfinished book should offer to mark it finished")
    assert(not H.findButton(unfinished.buttons, "Mark as unfinished"))

    local finishedBook = {id = "a1", library = "books", title = "Dune", authors = {"Frank Herbert"}, format = "epub", finished = true}
    plugin:showBook(finishedBook)
    local finished = H.shown[#H.shown]
    assert(H.findButton(finished.buttons, "Mark as unfinished"), "a finished book should offer to mark it unfinished")
    assert(not H.findButton(finished.buttons, "Mark as finished"))
end)

test("tapping the mark-finished toggle calls the Bearer-gated finished endpoint and confirms", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{finished = true} end
    plugin:showBook(dune)
    local dialog = H.shown[#H.shown]
    H.findButton(dialog.buttons, "Mark as finished").callback()
    local request = H.requests[#H.requests]
    assert(request.method == "PUT" and request.url:find("/finished/books/a1", 1, true), "must PUT to the finished endpoint")
    eq(H.decode(request.body).finished, true)
    assert(H.shown[#H.shown].text:find("inished", 1, true), H.shown[#H.shown].text)
end)

test("the book dialog offers a status chooser, because that is what parks a book", function()
    local plugin = H.newPlugin()
    plugin:showBook(dune)
    local dialog = H.shown[#H.shown]
    assert(H.findButton(dialog.buttons, "Set status…"), "the dialog should offer the status chooser")
end)

test("choosing a status PUTs to the device status endpoint and shows it afterwards", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{status = "on_hold"} end
    local book = {id = "a1", library = "books", title = "Dune", authors = {"Frank Herbert"}, format = "epub"}
    plugin:showStatusChoices(book)
    local chooser = H.shown[#H.shown]
    H.findButton(chooser.buttons, "On hold").callback()
    local request = H.requests[#H.requests]
    assert(request.method == "PUT" and request.url:find("/status/books/a1", 1, true), "must PUT to the status endpoint")
    eq(H.decode(request.body).status, "on_hold")
    eq(book.status, "on_hold")
    -- A parked book must say so the next time its dialog is opened.
    plugin:showBook(book)
    local reopened = H.shown[#H.shown]
    assert(reopened.title:find("On hold", 1, true), reopened.title)
end)

test("the settings page offers the parked shelf, and it asks only for parked books", function()
    local plugin = H.newPlugin()
    plugin:showSettingsPage()
    local page = H.shown[#H.shown]
    assert(rowNamed(page, "On hold and dropped"), "the Library section should offer the parked shelf")
end)

test("the book dialog shows the cover, fetched once and cached", function()
    local plugin = H.newPlugin()
    local book = {id = "a1", library = "books", title = "Dune", format = "epub", authors = {"Frank Herbert"}}
    H.respond = function() return 200, "\255\216\255\224cover-bytes" end
    local cover = plugin:coverFile(book, true)
    assert(cover, "the cover should be cached")
    assert(H.fileSize(cover) > 0, "the cached cover should not be empty")
    local before = #H.requests
    assert(plugin:coverFile(book, true) == cover, "the cached cover should be returned")
    assert(#H.requests == before, "a cached cover must not be fetched again")
end)

test("a cover that is really an HTML error page is not cached", function()
    local plugin = H.newPlugin()
    local book = {id = "html-not-a-cover", library = "books", title = "Dune", format = "epub"}
    H.respond = function() return 200, "<html><body>login</body></html>" end
    assert(plugin:coverFile(book, true) == nil, "an HTML body is not a cover")
    assert(H.fileSize(plugin:coverTarget(book)) == 0, "nothing should be cached")
end)

test("a book in a series offers the next volume, and tapping it opens that book", function()
    local plugin = H.newPlugin()
    local book = {id = "a1", library = "books", title = "Dune", series = "Dune", series_index = 1, format = "epub"}
    H.respond = function(request)
        if request.url and request.url:find("/series/", 1, true) then
            return 200, H.encode{series = "Dune", index = 1, next = {id = "a2", library = "books", title = "Dune Messiah", series = "Dune", series_index = 2, format = "epub"}}
        end
        return 404, ""
    end
    plugin:showBook(book)
    local dialog = H.shown[#H.shown]
    local next_row = H.findButton(dialog.buttons, "Next: Dune Messiah  ·  #2")
    assert(next_row, "the dialog should offer the next volume")
    next_row.callback()
    local opened = H.shown[#H.shown]
    assert(opened.title and opened.title:find("Dune Messiah", 1, true), "tapping it should open the next book")
    -- Twice through the same book must not ask the server twice.
    local asked = 0
    for _, request in ipairs(H.requests) do if tostring(request.url):find("/series/", 1, true) then asked = asked + 1 end end
    plugin:showBook(book)
    local after = 0
    for _, request in ipairs(H.requests) do if tostring(request.url):find("/series/", 1, true) then after = after + 1 end end
    assert(after == asked, "the answer should be kept on the book table")
end)

test("a book outside a series asks nothing about one", function()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:showBook({id = "n1", library = "books", title = "Neuromancer", format = "epub"})
    for _, request in ipairs(H.requests) do
        assert(not tostring(request.url):find("/series/", 1, true), "a standalone book should not ask the series route")
    end
end)

test("a book the server has no cover for is not asked again on every open", function()
    local plugin = H.newPlugin()
    local book = {id = "no-artwork", library = "books", title = "Emma", format = "epub"}
    H.respond = function() return 404, "" end
    assert(plugin:coverFile(book, true) == nil, "no cover means no cover")
    local asked = #H.requests
    assert(plugin:coverFile(book, true) == nil, "still no cover")
    assert(#H.requests == asked, "the miss should be remembered, not re-asked")
end)

test("the dialog carries the cover above the title, and falls back to plain text without one", function()
    local plugin = H.newPlugin()
    local book = {id = "a1", library = "books", title = "Dune", format = "epub", authors = {"Frank Herbert"}}
    H.respond = function() return 200, "\255\216\255\224cover-bytes" end
    plugin:showBook(book)
    local dialog = H.shown[#H.shown]
    assert(H.findValue(dialog, "file") == plugin:coverTarget(book), "the cover should be in the dialog")
    assert(H.findText(dialog, "Dune"), "the title should still be there")
    assert(H.findText(dialog, "Close"), "the buttons should still be there")

    -- A book whose cover the server does not have keeps the plain dialog it always had.
    H.shown = {}
    H.respond = function() return 404, "" end
    plugin:showBook({id = "a2", library = "books", title = "Emma", format = "epub"})
    local plain = H.shown[#H.shown]
    assert(type(plain.title) == "string" and plain.title:find("Emma", 1, true), "the plain dialog should still open")
end)

test("the parked shelf lists a held book with its status and opens it", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{total = 1, continue = {{book = {id = "a1", library = "books", title = "Dune", status = "on_hold", authors = {"Frank Herbert"}, format = "epub"}, title = "Dune", fraction = .4}}} end
    plugin:showParked()
    local shelf = H.shown[#H.shown]
    local row = rowNamed(shelf, "Dune")
    assert(row, "the parked book should be listed")
    assert(row.text:find("On hold", 1, true), row.text)
    row.callback()
    assert(H.shown[#H.shown].buttons, "tapping a parked book should open its dialog")
end)

test("a status the server refuses is reported, not silently swallowed", function()
    local plugin = H.newPlugin()
    H.respond = function() return 400, "" end
    plugin:setBookStatus({id = "a1", library = "books", title = "Dune"}, "on_hold")
    assert(H.shown[#H.shown].text:find("ould not", 1, true), H.shown[#H.shown].text)
end)

test("a failed mark-finished toggle says what happened", function()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:showBook(dune)
    local dialog = H.shown[#H.shown]
    H.findButton(dialog.buttons, "Mark as finished").callback()
    assert(H.shown[#H.shown].text:find("ould not", 1, true), H.shown[#H.shown].text)
end)

test("Stream closes the dialog and opens a fullscreen ImageViewer sized to the page count", function()
    local plugin = H.newPlugin()
    plugin:showBook(streamableComic)
    local dialog = H.shown[#H.shown]
    local closedBefore = #H.closed
    H.findButton(dialog.buttons, "Stream").callback()
    eq(#H.closed, closedBefore + 1, "the book dialog should close")
    local viewer = H.shown[#H.shown]
    eq(viewer.fullscreen, true)
    eq(viewer.images_list_nb, 5)
end)

test("the streamed page table fetches each page from the Bearer-gated koreader pages endpoint", function()
    local plugin = H.newPlugin()
    H.respond = function(req) return 200, "PAGE-BYTES" end
    plugin:streamBook(streamableComic)
    local viewer = H.shown[#H.shown]
    local page = viewer.image[1]
    local request = lastRequestFor("/api/koreader/v1/pages/manga/comic/0")
    assert(request, "page 1 should fetch index 0 from the new endpoint")
    assert(request.headers.Authorization:find("Bearer", 1, true), "the page fetch must use the device token")
    assert(page and page._rendered_from == "PAGE-BYTES", "the fetched bytes should be handed to RenderImage")
end)

test("a failed page fetch falls back to the placeholder image instead of crashing", function()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:streamBook(streamableComic)
    local viewer = H.shown[#H.shown]
    local page = viewer.image[1]
    assert(page and page._rendered_from_file, "a failed fetch should render the placeholder file")
end)

test("a failed page fetch notifies the reader instead of failing silently", function()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:streamBook(streamableComic)
    local viewer = H.shown[#H.shown]
    local shownBefore = #H.shown
    local _ = viewer.image[1]
    local notice = H.shown[#H.shown]
    assert(#H.shown > shownBefore and notice.text and notice.text:find("failed", 1, true), "a failed page fetch must surface a visible notice")
end)

test("a failed page fetch only notifies once per page, not on every repeated access", function()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:streamBook(streamableComic)
    local viewer = H.shown[#H.shown]
    local _ = viewer.image[1]
    local shownAfterFirst = #H.shown
    local _ = viewer.image[1]
    eq(#H.shown, shownAfterFirst, "repeated access to the same failed page must not spam notices")
end)

test("streaming resumes at the book's last-read page when known, otherwise opens at page 1", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "PAGE-BYTES" end
    plugin:streamBook({id = "comic", library = "manga", title = "Comic", format = "cbz", streamable = true, page_count = 5, last_page = 4})
    eq(H.shown[#H.shown]._shown_image_num, 4, "known progress should resume at that page")

    plugin:streamBook(streamableComic)
    eq(H.shown[#H.shown]._shown_image_num, 1, "unknown progress opens at page 1")
end)

test("downloads are laid out as Series/<series>/<NN - title> and <author>/<title>, with safe names", function()
    local root = "/books"
    local plugin = H.newPlugin()
    local P = getmetatable(plugin).__index
    eq(P.downloadPath(root, dune, "series"), "/books/Series/Dune/01 - Dune.epub")
    eq(P.downloadPath(root, {title = "Half", authors = {"A"}, series = "S", series_index = 1.5, format = "epub"}, "series"), "/books/Series/S/1.5 - Half.epub")
    eq(P.downloadPath(root, neuromancer, "series"), "/books/William Gibson/Neuromancer.epub")
    eq(P.downloadPath(root, dune, "author"), "/books/Frank Herbert/Dune.epub")
    eq(P.downloadPath(root, dune, "flat"), "/books/Dune.epub")
    eq(P.downloadPath(root, {title = "What: a/b?", authors = {}, format = "cbz"}, "series"), "/books/Unknown author/What_ a_b_.cbz")
    eq(P.cleanName("  ..hidden.. "), "hidden")
    eq(P.cleanName(""), "Untitled")
    assert(#P.cleanName(string.rep("é", 200)) <= 100, "long names are cut on a character boundary")
end)

test("the download queue fetches one book at a time, in order, and keeps going past a failure", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local messiah = {id = "a2", library = "books", title = "Dune Messiah", series = "Dune", series_index = 2, format = "epub", size = 12}
    local downloads = function()
        local out = {}
        for _, request in ipairs(H.requests) do
            if tostring(request.url):find("/koreader/v1/download/", 1, true) then out[#out + 1] = tostring(request.url) end
        end
        return out
    end
    local respond_all = function(req)
        if req.url:find("/koreader/v1/download/", 1, true) then return 200, "EPUB BYTES" end
        return 404, "{}"
    end
    H.respond = respond_all
    assert(plugin:queueBook(dune) and plugin:queueBook(messiah))
    eq(#downloads(), 0, "nothing is transferred until the waiting message has been drawn")
    eq(plugin:queueList()[1].state, "downloading", "the first book is the one in flight")
    eq(plugin:queueList()[2].state, "waiting", "the second waits its turn")
    H.scheduled[#H.scheduled].fn()
    eq(#downloads(), 1, "only the first book is fetched")
    eq(plugin:queueList()[2].state, "downloading", "finishing the first starts the second")
    H.scheduled[#H.scheduled].fn()
    eq(#downloads(), 2)
    assert(H.read(root .. "/Series/Dune/01 - Dune.epub") == "EPUB BYTES", "the first book is on disk")
    assert(H.read(root .. "/Series/Dune/02 - Dune Messiah.epub") == "EPUB BYTES", "the second book is on disk")
    eq(plugin:queueList()[1].state, "done")
    eq(plugin:queueList()[2].state, "done")

    -- A book the server refuses must not stop the ones behind it.
    local plugin2 = H.newPlugin()
    local broken = {id = "gone", library = "books", title = "Gone", format = "epub", size = 10}
    -- A fresh book: the two above are already on disk, and queueing those is correctly refused.
    local children = {id = "a3", library = "books", title = "Children of Dune", series = "Dune", series_index = 3, format = "epub", size = 12}
    H.respond = function(req)
        if req.url:find("/download/books/gone", 1, true) then return 404, "" end
        if req.url:find("/koreader/v1/download/", 1, true) then return 200, "EPUB BYTES" end
        return 404, "{}"
    end
    assert(plugin2:queueBook(broken) and plugin2:queueBook(children))
    H.scheduled[#H.scheduled].fn()
    eq(plugin2:queueList()[1].state, "failed", "the bad book is marked failed")
    eq(plugin2:queueList()[2].state, "downloading", "and the queue moved on to the next one")
    eq(#downloads(), 3)
    H.scheduled[#H.scheduled].fn()
    eq(plugin2:queueList()[2].state, "done")
end)

test("a queued book can be taken out again before it starts", function()
    local plugin = H.newPlugin()
    local messiah = {id = "a4", library = "books", title = "Heretics of Dune", series = "Dune", series_index = 5, format = "epub", size = 12}
    H.respond = function(req)
        if req.url:find("/koreader/v1/download/", 1, true) then return 200, "EPUB BYTES" end
        return 404, "{}"
    end
    local firstBook = {id = "a5", library = "books", title = "Chapterhouse: Dune", series = "Dune", series_index = 6, format = "epub", size = 12}
    assert(plugin:queueBook(firstBook) and plugin:queueBook(messiah))
    local menus = 0
    for _, widget in ipairs(H.shown) do
        if widget.title == "Download queue" then menus = menus + 1 end
    end
    eq(menus, 1, "queueing twice must not stack two queue windows")
    local items = plugin.queue_widget.item_table
    assert(items[2].text:find("waiting", 1, true), items[2].text)
    items[2].callback()
    eq(#plugin:queueList(), 1, "tapping a waiting book removes it")
    H.scheduled[#H.scheduled].fn()
    local downloads = 0
    for _, request in ipairs(H.requests) do
        if tostring(request.url):find("/koreader/v1/download/", 1, true) then downloads = downloads + 1 end
    end
    eq(downloads, 1, "the removed book is never fetched")
end)

test("the queue says how much has arrived, in bytes", function()
    local Q = getmetatable(H.newPlugin()).__index
    eq(Q.queueProgressText(1048576, 3145728), "1.0 MB of 3.0 MB")
    eq(Q.queueProgressText(0, 0), "0.0 MB")
end)

test("downloading saves the book where the layout says, with the device token, and offers to open it", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function(req)
        if req.url:find("/download/books/a1", 1, true) then return 200, "EPUB BYTES" end
        return 404, "{}"
    end
    plugin:showBook(dune)
    local dialog = H.shown[#H.shown]
    H.findButton(dialog.buttons, "Download").callback()
    assert(#H.scheduled > 0, "the download starts after the message has been drawn")
    H.scheduled[#H.scheduled].fn()
    local path = root .. "/Series/Dune/01 - Dune.epub"
    eq(H.read(path), "EPUB BYTES")
    eq(H.read(path .. ".part"), nil)
    local request = lastRequestFor("/api/koreader/v1/download/books/a1")
    assert(request.headers.Authorization:find("Bearer", 1, true), "the download uses the device token")
    local confirm = H.shown[#H.shown]
    assert(confirm.text:find(path, 1, true), "it says where the book went")
    confirm.ok_callback()
    eq(H.opened[1], path)
    -- the same book is now marked, and offers Open rather than another download
    plugin:showBook(dune)
    local again = H.shown[#H.shown]
    assert(H.findButton(again.buttons, "Open") and not H.findButton(again.buttons, "Download"), "a downloaded book offers Open")
end)

test("an owned download is reused after its catalogue title changes", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local book = {id="one", library="books", title="Old title", authors={"Writer"}, format="epub"}
    H.respond = function() return 200, "OWNED" end
    plugin:downloadBook(book)
    H.scheduled[#H.scheduled].fn()
    book.title = "New title"
    plugin:openBook(book)
    eq(H.opened[1], root .. "/Writer/Old title.epub", "existing owned file must open")
    eq(#H.requests, 1, "must not redownload")
end)

test("an owned collision remains reusable when the earlier file disappears", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local book = {id="one", library="books", title="Same", authors={"Writer"}, format="epub"}
    os.execute("mkdir -p '" .. root .. "/Writer'")
    local unowned = root .. "/Writer/Same.epub"
    H.write(unowned, "UNOWNED")
    H.respond = function() return 200, "OWNED" end
    plugin:downloadBook(book)
    H.scheduled[#H.scheduled].fn()
    os.remove(unowned)
    plugin:openBook(book)
    eq(H.opened[1], root .. "/Writer/Same (1).epub")
    eq(#H.requests, 1)
end)

test("two books with the same filename never open one another", function()
    downloadFolder()
    local plugin = H.newPlugin()
    local one = {id="one", library="books", title="Same", authors={"Writer"}, format="epub"}
    local two = {id="two", library="books", title="Same", authors={"Writer"}, format="epub"}
    H.respond = function() return 200, "ONE" end
    plugin:downloadBook(one)
    H.scheduled[#H.scheduled].fn()
    plugin:openBook(two)
    eq(#H.opened, 0, "opened the first book for a different catalogue entry")
    assert(H.findButton(H.shown[#H.shown].buttons, "Download"))
end)

test("identical catalogue ids on different servers never open the other server's book", function()
    downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "SERVER ONE" end
    plugin:downloadBook(neuromancer)
    H.scheduled[#H.scheduled].fn()
    require("mochil_config").server_url = "https://different-server.test"
    plugin:openBook(neuromancer)
    eq(#H.opened, 0, "opened a book owned by a different server")
    assert(H.findButton(H.shown[#H.shown].buttons, "Download"))
end)

test("unowned and legacy-owned files are left untouched rather than claimed", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local dir = root .. "/William Gibson"
    os.execute("mkdir -p '" .. dir .. "'")
    local path = dir .. "/Neuromancer.epub"
    H.write(path, "ORIGINAL")
    plugin:openBook(neuromancer)
    eq(#H.opened, 0)
    assert(H.findButton(H.shown[#H.shown].buttons, "Download"))
    G_reader_settings:saveSetting("mochil_download_owners", {[path]="books\0n1"})
    H.respond = function() return 200, "NEW" end
    plugin:downloadBook(neuromancer)
    H.scheduled[#H.scheduled].fn()
    eq(H.read(path), "ORIGINAL")
    eq(H.read(dir .. "/Neuromancer (1).epub"), "NEW")
end)

test("an unowned partial file is never overwritten", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local dir = root .. "/William Gibson"
    os.execute("mkdir -p '" .. dir .. "'")
    local part = dir .. "/Neuromancer.epub.part"
    H.write(part, "UNOWNED PART")
    H.respond = function() return 200, "NEW" end
    plugin:downloadBook(neuromancer)
    H.scheduled[#H.scheduled].fn()
    eq(H.read(part), "UNOWNED PART")
end)

test("an empty 200 response is rejected instead of being saved and owned as a book", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "" end
    plugin:downloadBook(neuromancer)
    H.scheduled[#H.scheduled].fn()
    eq(H.read(root .. "/William Gibson/Neuromancer.epub"), nil, "an empty body must never be saved as a book")
    local owners = G_reader_settings:readSetting("mochil_download_owners") or {}
    local owned = false
    for _, owner in pairs(owners) do if owner == "books\0neuromancer" then owned = true end end
    eq(owned, false, "an empty body must never be recorded as an owned download")
    assert(H.shown[#H.shown].text:find("empty", 1, true) or H.shown[#H.shown].text:find("no data", 1, true), "the failure must say the download was empty")
end)

test("Back up all highlights saves the export to a file in the download folder", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "EXPORTED-JSON" end
    plugin:backupHighlights()
    assert(fired("/api/koreader/v1/annotations/export"), "the export endpoint must be requested")
    local message = H.shown[#H.shown]
    local path = message.text:match("(/%S+%.json)")
    assert(path, "the confirmation should name the saved file: " .. tostring(message.text))
    eq(H.read(path), "EXPORTED-JSON")
end)

test("a failed highlight backup says what happened and saves nothing", function()
    downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:backupHighlights()
    assert(H.shown[#H.shown].text:find("ould not", 1, true), H.shown[#H.shown].text)
end)

test("Export highlights to Notes posts to the book's notes endpoint and confirms", function()
    local plugin = H.newPlugin()
    H.respond = function(req) return 201, H.encode{slug = "annotations-a-book"} end
    plugin:exportHighlightsToNotes(true)
    local request = H.requests[#H.requests]
    assert(request.url:find("/notes", 1, true), "must post to the notes endpoint")
    assert(request.method == "POST")
    assert(H.shown[#H.shown].text:find("xported", 1, true), H.shown[#H.shown].text)
end)

test("a failed Notes export says what happened", function()
    local plugin = H.newPlugin()
    H.respond = function() return 409, "" end
    plugin:exportHighlightsToNotes(true)
    assert(H.shown[#H.shown].text:find("ould not", 1, true), H.shown[#H.shown].text)
end)

test("the notes & wiki menu offers an enable toggle, and hides the read action until turned on", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_wiki_enabled", nil)
    local function wikiMenu(entries)
        for _, entry in ipairs(entries) do if entry.text == "Notes & wiki" then return entry end end
    end
    local function findItem(subItems, fragment)
        for _, item in ipairs(subItems) do if item.text and item.text:find(fragment, 1, true) then return item end end
    end
    local notes = wikiMenu(plugin:menuEntries())
    assert(notes, "a Notes & wiki menu entry must exist")
    local toggle = findItem(notes.sub_item_table, "Enable")
    assert(toggle, "an enable toggle must exist")
    assert(not toggle.checked_func(), "must start disabled")
    assert(not findItem(notes.sub_item_table, "Read my notes"), "the read action must not appear until enabled")

    toggle.callback()
    assert(toggle.checked_func(), "toggling must turn it on")
    notes = wikiMenu(plugin:menuEntries())
    assert(findItem(notes.sub_item_table, "Read my notes"), "the read action must appear once enabled")

    toggle.callback()
    assert(not toggle.checked_func(), "toggling again must turn it off")
    notes = wikiMenu(plugin:menuEntries())
    assert(not findItem(notes.sub_item_table, "Read my notes"), "the read action must disappear once disabled again")
    G_reader_settings:saveSetting("mochil_wiki_enabled", nil)
end)

test("Read my notes on this device downloads and opens the rendered HTML", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "<html>MY NOTES</html>" end
    plugin:readWikiOnDevice()
    assert(fired("/api/koreader/v1/wiki.html"), "the wiki.html endpoint must be requested")
    assert(#H.opened == 1, "the rendered notes must be opened on-device, not just saved")
    local path = H.opened[1]
    assert(path:find("%.html$"), "the opened file must be an .html document: " .. tostring(path))
    eq(H.read(path), "<html>MY NOTES</html>")
end)

test("a failed notes read says what happened and opens nothing", function()
    downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 404, "" end
    plugin:readWikiOnDevice()
    eq(#H.opened, 0)
    assert(H.shown[#H.shown].text:find("ould not", 1, true), H.shown[#H.shown].text)
end)

test("reading a vault downloads its rendered HTML directly when only one vault exists", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function(req)
        if req.url:find("/home", 1, true) then return 200, H.encode{vaults = {{id = "wiki", name = "My notes"}}} end
        if req.url:find("/vault/wiki.html", 1, true) then return 200, "<html>VAULT NOTES</html>" end
        return 404, ""
    end
    plugin:readVaultOnDevice()
    assert(#H.opened == 1, "the rendered vault must be opened on-device")
    local path = H.opened[1]
    assert(path:find("%.html$"), path)
    eq(H.read(path), "<html>VAULT NOTES</html>")
end)

test("reading a vault offers a picker when several vaults exist", function()
    downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function(req)
        if req.url:find("/home", 1, true) then
            return 200, H.encode{vaults = {{id = "wiki", name = "Notes"}, {id = "journal", name = "Journal"}}}
        end
        if req.url:find("/vault/journal.html", 1, true) then return 200, "<html>JOURNAL</html>" end
        return 404, ""
    end
    plugin:readVaultOnDevice()
    local picker = H.shown[#H.shown]
    assert(H.findButton(picker.buttons, "Journal"), "both vaults should be offered: " .. pageText(picker))
    H.findButton(picker.buttons, "Journal").callback()
    eq(H.read(H.opened[1]), "<html>JOURNAL</html>")
end)

test("reading a vault with none configured says so and opens nothing", function()
    downloadFolder()
    local plugin = H.newPlugin()
    H.respond = function() return 200, H.encode{} end
    plugin:readVaultOnDevice()
    eq(#H.opened, 0)
    assert(H.shown[#H.shown].text:find("o vault", 1, true), H.shown[#H.shown].text)
end)

test("a failed download leaves no file behind and says what happened", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    for _, case in ipairs{{404, "no longer has"}, {401, "expired"}} do
        H.respond = function() return case[1], "nope" end
        plugin:showBook(neuromancer)
        H.findButton(H.shown[#H.shown].buttons, "Download").callback()
        H.scheduled[#H.scheduled].fn()
        local message = H.shown[#H.shown]
        assert(message.text:find(case[2], 1, true), message.text)
        eq(H.read(root .. "/William Gibson/Neuromancer.epub"), nil)
        eq(H.read(root .. "/William Gibson/Neuromancer.epub.part"), nil)
    end
    H.respond = function() return nil, "connection refused" end
    plugin:showBook(neuromancer)
    H.findButton(H.shown[#H.shown].buttons, "Download").callback()
    H.scheduled[#H.scheduled].fn()
    assert(H.shown[#H.shown].text:find("refused", 1, true), H.shown[#H.shown].text)
end)

test("a downloaded book is marked in the list", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    os.execute("mkdir -p '" .. root .. "/William Gibson'")
    H.write(root .. "/William Gibson/Neuromancer.epub", "x")
    G_reader_settings:saveSetting("mochil_download_owners", {[root .. "/William Gibson/Neuromancer.epub"]=require("ffi/sha2").md5("https://books.test") .. "\0books\0n1"})
    H.respond = browseServer{["kind=books"] = {kind = "books", total = 2, books = {dune, neuromancer}}}
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    eq(row(list, "Neuromancer").mandatory, "✓")
    eq(row(list, "Dune").mandatory, "EPUB")
end)

test("opening a book clears the browsing pages away first", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    H.respond = browseServer{["kind=books"] = {kind = "books", total = 1, books = {neuromancer}}}
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    plugin:openDownloaded(root .. "/x.epub")
    local closed = false
    for _, widget in ipairs(H.closed) do if widget == list then closed = true end end
    assert(closed, "the list must not stay open over the reader")
    eq(H.opened[#H.opened], root .. "/x.epub")
end)

test("the download folder and layout are settings; the folder picker saves the choice", function()
    local plugin = H.newPlugin()
    G_reader_settings:saveSetting("mochil_download_dir", nil)
    G_reader_settings:saveSetting("home_dir", "/sdcard/Books/")
    eq(getmetatable(plugin).__index.downloadRoot(), "/sdcard/Books/Mochil")
    local entries = plugin:menuEntries()
    local folder
    for _, entry in ipairs(entries) do if entry.text_func and entry.text_func():find("Download folder", 1, true) then folder = entry end end
    assert(folder, "the settings must show the download folder")
    folder.callback()
    local picker = H.shown[#H.shown]
    eq(picker.select_directory, true)
    picker.onConfirm("/mnt/books/")
    eq(getmetatable(plugin).__index.downloadRoot(), "/mnt/books")
    local layout
    for _, entry in ipairs(entries) do if entry.text == "Download layout" then layout = entry end end
    assert(layout and #layout.sub_item_table == 3, "three layouts to choose from")
    layout.sub_item_table[3].callback()
    eq(getmetatable(plugin).__index.downloadLayout(), "flat")
    layout.sub_item_table[1].callback()
    eq(getmetatable(plugin).__index.downloadLayout(), "series")
    G_reader_settings:saveSetting("home_dir", nil)
end)

-- ===== Continue reading opens the book =====
local function findAll(widget, predicate, found, seen)
    found, seen = found or {}, seen or {}
    if type(widget) ~= "table" or seen[widget] then return found end
    seen[widget] = true
    if predicate(widget) then found[#found + 1] = widget end
    for key, child in pairs(widget) do if key ~= "show_parent" then findAll(child, predicate, found, seen) end end
    return found
end
local continueHome = function(entries)
    return function() return 200, H.encode{account = "kevin", books = 1, series = 0, authors = 1, collections = 0, continue = entries} end
end
local readingBook = {id = "n1", library = "books", title = "Neuromancer", authors = {"William Gibson"}, format = "epub", size = 1048576}

test("both home styles qualify generic chapters without changing edition identity", function()
    for _, plain in ipairs{true, false} do
        local plugin = H.newPlugin()
        local opened = {}
        plugin.openBook = function(_, book) opened[#opened + 1] = book.id end
        local series = "A very long manga series name that should remain part of the title"
        local page = openHome(plugin, continueHome{
            {title=" Chapter 0 ", authors={"Writer"}, book={id="a", library="manga", series=series, read_along=true}},
            {title="Chapter 0", authors={"Writer"}, book={id="b", library="manga", series=series}},
            {title="Chapter 2", authors={}, book={id="c", library="manga"}},
            {title="Aku no Kyouten — Chapter 3", authors={}, book={id="d", library="manga", series="Aku no Kyouten"}},
        }, plain)
        local targets
        if plain then
            targets = {}; for _, item in ipairs(page.item_table) do if item.callback and (item.text:find("Chapter", 1, true)) then targets[#targets + 1] = item end end
        else targets = findAll(page, function(w) return type(w.onTapCard) == "function" end) end
        eq(#targets, 4)
        local texts = {}
        for _, item in ipairs(targets) do
            local text = plain and item.text or table.concat(H.widgetText(item), " ")
            texts[#texts + 1] = text
            if plain then item.callback() else item.onTapCard() end
        end
        local all = table.concat(texts, "\n")
        assert(all:find(series .. " — Chapter 0", 1, true), all)
        assert(all:find("Read-along", 1, true), all)
        assert(all:find("Chapter 2", 1, true), all)
        assert(all:find("Aku no Kyouten — Chapter 3", 1, true), all)
        assert(not all:find("Aku no Kyouten — Aku no Kyouten", 1, true), all)
        table.sort(opened); eq(table.concat(opened, ","), "a,b,c,d")
    end
end)

test("tapping Continue reading offers a download, or opens the book when it is already here (both page styles)", function()
    for _, plain in ipairs{true, false} do
        local root = downloadFolder()
        local plugin = H.newPlugin()
        local page = openHome(plugin, continueHome{{title = "Neuromancer", authors = {"William Gibson"}, fraction = 0.3, book = readingBook}}, plain)
        local function tap()
            if plain then row(page, "Neuromancer").callback()
            else
                local cards = findAll(page, function(w) return type(w.onTapCard) == "function" end)
                eq(#cards, 1)
                assert(cards[1].ges_events.TapCard, "the card must listen for taps")
                assert(cards[1].onTapCard() == true, "the tap is handled")
            end
        end
        tap()
        local dialog = H.shown[#H.shown]
        assert(dialog.buttons and H.findButton(dialog.buttons, "Download"), "a book that is not here yet offers Download")
        os.execute("mkdir -p '" .. root .. "/William Gibson'")
        H.write(root .. "/William Gibson/Neuromancer.epub", "x")
        G_reader_settings:saveSetting("mochil_download_owners", {[root .. "/William Gibson/Neuromancer.epub"]=require("ffi/sha2").md5("https://books.test") .. "\0books\0n1"})
        tap()
        eq(H.opened[#H.opened], root .. "/William Gibson/Neuromancer.epub")
    end
end)

test("same-title editions show and open their own downloads in both home styles", function()
    for _, plain in ipairs{true, false} do
        local root = downloadFolder()
        H.opened, H.requests, H.scheduled = {}, {}, {}
        local plugin = H.newPlugin()
        local a = {id="a", library="books", title="Same", authors={"Writer"}, format="epub"}
        local b = {id="b", library="books", title="Same", authors={"Writer"}, format="epub", read_along=true}
        H.respond = function(req) return 200, req.url:find("/a", 1, true) and "A" or "B" end
        plugin:downloadBook(a); H.scheduled[#H.scheduled].fn()
        plugin:downloadBook(b); H.scheduled[#H.scheduled].fn()
        local page = openHome(plugin, continueHome{
            {title="Same", authors={"Writer"}, book=a}, {title="Same", authors={"Writer"}, book=b},
        }, plain)
        local targets
        if plain then
            targets = {}; for _, item in ipairs(page.item_table) do if item.text:find("Same", 1, true) then targets[#targets+1] = item end end
        else targets = findAll(page, function(w) return type(w.onTapCard) == "function" end) end
        eq(#targets, 2)
        local badgeCount = 0
        for _, item in ipairs(targets) do
            local text = plain and item.text or table.concat(H.widgetText(item), " ")
            if text:find("Read.along") then badgeCount = badgeCount + 1 end
            if plain then item.callback() else item.onTapCard() end
            local expected = text:find("Read.along") and "/Writer/Same (1).epub" or "/Writer/Same.epub"
            eq(H.opened[#H.opened], root .. expected, "each edition opens its own file")
        end
        eq(badgeCount, 1)
        table.sort(H.opened)
        eq(H.opened[1], root .. "/Writer/Same (1).epub")
        eq(H.opened[2], root .. "/Writer/Same.epub")
        eq(#H.requests, 3, "two downloads and the home request only")
    end
end)

test("an entry the server did not name is shown but not tappable", function()
    local plugin = H.newPlugin()
    local plain = openHome(plugin, continueHome{{title = "Some Audiobook", authors = {}, fraction = 0.5}}, true)
    eq(row(plain, "Some Audiobook").callback, nil)
    local designed = openHome(plugin, continueHome{{title = "Some Audiobook", authors = {}, fraction = 0.5}}, false)
    eq(#findAll(designed, function(w) return type(w.onTapCard) == "function" end), 0)
end)

test("a card's tap area is where it was drawn", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, continueHome{{title = "Neuromancer", authors = {}, fraction = 0.3, book = readingBook}}, false)
    local card = findAll(page, function(w) return type(w.onTapCard) == "function" end)[1]
    card.dimen = {x = 10, y = 20, w = 100, h = 50}
    eq(card.ges_events.TapCard[1].range(), card.dimen)
end)

-- ===== choosing a library first =====
local lightNovels = {name = "Light novels", library = "light-novels", id = "light-novels", count = 120, series = 14, authors = 30, collections = 2}
local booksLibrary = {name = "Books", library = "books", id = "books", count = 5000, series = 900, authors = 1000}
local function libraryHome(libraries)
    return function(req)
        if req.url:find("kind=libraries", 1, true) then return 200, H.encode{kind = "libraries", total = #libraries, groups = libraries} end
        if req.url:find("/home", 1, true) then return 200, H.encode{account = "kevin", books = 5120, libraries = #libraries, series = 914, authors = 1030, collections = 2, continue = {}} end
        return 200, H.encode{kind = "books", total = 0, books = {}}
    end
end

test("Library opens the list of libraries, and a library offers books, series and authors", function()
    for _, plain in ipairs{true, false} do
        local plugin = H.newPlugin()
        local page = openHome(plugin, libraryHome{booksLibrary, lightNovels}, plain)
        if plain then row(page, "Libraries").callback()
        else
            local tile = H.findButton(page, "Libraries   2")
            assert(tile, "the tile counts libraries, not books")
            tile.callback()
        end
        local list = H.shown[#H.shown]
        assert(lastRequestFor("kind=libraries"), "the list of libraries is asked for")
        eq(row(list, "Light novels").mandatory, "120")
        row(list, "Light novels").callback()
        local library = H.shown[#H.shown]
        eq(row(library, "All books").mandatory, "120")
        eq(row(library, "Series").mandatory, "14")
        eq(row(library, "Authors").mandatory, "30")
        eq(row(library, "Collections").mandatory, "2")
        row(library, "Series").callback()
        assert(lastRequestFor("kind=series") and lastRequestFor("library=light-novels"), "series are limited to the chosen library")
        row(library, "Authors").callback()
        assert(lastRequestFor("kind=authors&library=light-novels") or (lastRequestFor("kind=authors") and lastRequestFor("library=light-novels")), "authors too")
        row(library, "All books").callback()
        assert(lastRequestFor("kind=books") and lastRequestFor("library=light-novels"), "books too")
    end
end)

test("a library page offers a Continue reading here filter scoped to just that library", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, libraryHome{booksLibrary, lightNovels}, true)
    row(page, "Libraries").callback()
    local list = H.shown[#H.shown]
    row(list, "Light novels").callback()
    local library = H.shown[#H.shown]
    local continueRow = row(library, "Continue reading here")
    assert(continueRow and continueRow.callback, "a library page must offer a scoped continue-reading entry")
    continueRow.callback()
    local continueList = H.shown[#H.shown]
    assert(lastRequestFor("library=light-novels") and lastRequestFor("/continue"), "the continue list must be filtered to this library")
    assert(tostring(continueList.title):find("Light novels", 1, true), "the scoped list should say which library it is")
end)

test("with one library there is nothing to choose, and empty collections are not offered", function()
    local plugin = H.newPlugin()
    local only = {name = "Books", library = "books", id = "books", count = 10, series = 2, authors = 3}
    local page = openHome(plugin, libraryHome{only}, true)
    row(page, "Libraries").callback()
    local library = H.shown[#H.shown]
    eq(row(library, "All books").mandatory, "10")
    local hasCollections = false
    for _, item in ipairs(library.item_table) do if item.text == "Collections" then hasCollections = true end end
    eq(hasCollections, false)
end)

test("a server that does not report libraries yet still gets the plain book list", function()
    local plugin = H.newPlugin()
    local page = openHome(plugin, browseServer{["kind=books"] = {kind = "books", total = 1, books = {dune}}}, true)
    row(page, "Books").callback()
    assert(pageText(H.shown[#H.shown]):find("Dune", 1, true), "the old behaviour remains")
end)

test("an answer that is not an object (a proxy page, a bare number) is refused instead of crashing the list", function()
    local plugin = H.newPlugin()
    H.respond = function() return 200, "42" end
    local body, reason = plugin:request("GET", nil, true, "browse?kind=books", true)
    eq(body, nil, "a number is not an answer")
    eq(reason, "invalid_response")
    plugin:showBrowse{kind = "books", title = "Library"}
    local list = H.shown[#H.shown]
    assert(list, "the list still opens, saying what went wrong")
    assert(row(list, "Try again"), "and offers to try again")
end)

test("browsing says it is loading while the server answers, and always takes the message away", function()
    local plugin = H.newPlugin()
    H.respond = browseServer{["kind=books"] = {kind = "books", total = 1, books = {dune}}}
    local shown, closed = #H.shown, #H.closed
    plugin:showBrowse{kind = "books", title = "Library"}
    local loading = H.shown[shown + 1]
    assert(loading and loading.text == "Loading…", "a Loading message is up before the list")
    local gone = false
    for index = closed + 1, #H.closed do if H.closed[index] == loading then gone = true end end
    assert(gone, "the message is closed once the answer is in")
    -- Even when the request itself fails the message must not stay on screen.
    H.respond = function() error("the network fell over") end
    local before = #H.closed
    plugin:showBrowse{kind = "series", title = "Series"}
    assert(#H.closed > before, "the loading message was closed after a failure")
end)

test("a book marked finished in KOReader is pushed as finished, and only then", function()
    local plugin, ui, settings = H.newPlugin()
    plugin:push(true)
    eq(H.decode(H.requests[1].body).finished, nil, "an ordinary push carries no flag")
    settings.summary = {status = "complete"}
    plugin:push(false)
    eq(#H.requests, 2, "marking the book finished must send even though the position did not move")
    eq(H.decode(H.requests[2].body).finished, true)
    plugin:push(false)
    eq(#H.requests, 2, "nothing new to say")
    settings.summary = {status = "reading"}
    plugin:push(false)
    eq(#H.requests, 3)
    eq(H.decode(H.requests[3].body).finished, nil, "no longer complete")
end)

-- A highlight this plugin uploads carries a koreader:<hash> id. Dropping those on import meant highlights made on one
-- reader never appeared on another, silently and for good.
test("a highlight uploaded from another reader is imported, not filtered out", function()
    local plugin, ui = H.newPlugin()
    local native = H.encode{pos0 = "/body/DocFragment[2]/p[1]", pos1 = "/body/DocFragment[2]/p[1]"}
    H.respond = function(req)
        if req.method == "GET" then
            return 200, H.encode{revision = 1, applied = {{id = "koreader:h71", revision = 1, quote = "a quote", locators = {koreader = native}}}}
        end
        return 200, H.encode{revision = 1, applied = {}}
    end
    plugin:syncAnnotations(false)
    eq(#ui.annotation.annotations, 1, "a highlight made on another reader must be imported")
    eq(ui.annotation.annotations[1].mochil_id, "koreader:h71")
end)

-- A deletion made elsewhere has to reach the highlight that was created here, which only has a derived id.
test("a highlight made here is removed when it is deleted elsewhere", function()
    local plugin, ui, settings = H.newPlugin()
    ui.annotation.annotations = {{drawer = "lighten", text = "x", pos0 = "/body/DocFragment[2]/p[1]", pos1 = "/body/DocFragment[2]/p[1]", datetime = "2026-01-01 10:00:00"}}
    local before = plugin:localAnnotations()
    local id = next(before)
    settings.mochil_annotation_snapshot = {[id] = {revision = 1, signature = before[id]._signature}}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 2, applied = {{id = id, deleted = true, revision = 2}}} end
        return 200, H.encode{revision = 2, applied = {}}
    end
    plugin:syncAnnotations(true)
    eq(#ui.annotation.annotations, 0, "the tombstone was not applied to the local highlight")
    plugin:syncAnnotations(true)
    for _, change in ipairs(H.decode(H.requests[#H.requests].body or "{}").changes or {}) do
        assert(not (change.annotation and change.annotation.id == id and change.annotation.deleted == false),
            "an applied tombstone was uploaded again and re-created the highlight")
    end
end)

-- On a PDF or CBZ applyRemoteAnnotations returns before setting up its bookkeeping, which used to abort the whole
-- sync with a Lua error the moment a remote id also existed locally.
test("highlight sync survives a fixed-layout book that has a local edit", function()
    local plugin, ui = H.newPlugin()
    ui.rolling = nil
    ui.document.info.has_pages = true
    ui.annotation.annotations = {{drawer = "lighten", text = "x", page = 5}}
    local before = plugin:localAnnotations()
    local id = next(before)
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 1, applied = {{id = id, revision = 1, quote = "x"}}} end
        return 200, H.encode{revision = 3, applied = {{id = id, revision = 3}}}
    end
    plugin:syncAnnotations(true)
    local sent = false
    for _, request in ipairs(H.requests) do if request.method == "PUT" then sent = true end end
    assert(sent, "the local edit on a fixed-layout book never left the device")
    eq(#H.logs, 0, "the sync reported an error")
end)

-- A staging folder left behind by an interrupted download (power off, killed reader) must not block that book for
-- good; the reader could only have fixed it by hand.
test("a staging folder left by an interrupted download does not block the book", function()
    local root = downloadFolder()
    local plugin = H.newPlugin()
    local book = {id="one", library="books", title="Later", authors={"Writer"}, format="epub"}
    os.execute("mkdir -p '" .. root .. "/Writer/Later.epub.part'")
    H.write(root .. "/Writer/Later.epub.part/download", "HALF")
    H.respond = function() return 200, "FULL" end
    plugin:downloadBook(book)
    H.scheduled[#H.scheduled].fn()
    eq(H.read(root .. "/Writer/Later.epub"), "FULL", "the leftover staging folder blocked the download")
    eq(H.read(root .. "/Writer/Later.epub.part/download"), nil)
end)

-- The device's own statistics database only sees this reader, so the server is what makes the panel cross-device.
test("the reading statistics panel shows the server's numbers", function()
    local plugin = H.newPlugin()
    H.respond = function(req)
        assert(req.url:find("/api/koreader/v1/stats", 1, true), "the stats endpoint must be requested: " .. req.url)
        return 200, H.encode{in_progress = 4, finished = 12, finished_this_month = 2, pages_read = 1840,
            recent_finished = {{library = "books", id = "a1", title = "Dune", finished_at = 1700000000000}}}
    end
    plugin:readingStats()
    local panel = H.shown[#H.shown]
    assert(panel.text:find("Books in progress: 4", 1, true), "missing the in-progress count: " .. tostring(panel.text))
    assert(panel.text:find("Finished this month: 2", 1, true), "missing this month's count")
    assert(panel.text:find("Pages read: 1840", 1, true), "missing the page total")
    assert(panel.text:find("Dune", 1, true), "missing the recently finished books")
end)

-- The statistics plugin is optional and its method names vary by release, so a device without it still gets the panel.
test("the reading statistics panel works without KOReader's statistics plugin", function()
    local plugin, ui = H.newPlugin()
    ui.statistics = {getTodayBookStats = function() error("not this release") end}
    H.respond = function() return 200, H.encode{in_progress = 1} end
    plugin:readingStats()
    local panel = H.shown[#H.shown]
    assert(panel.text:find("Books in progress: 1", 1, true), "the server's numbers must still be shown")
    assert(not panel.text:find("On this reader", 1, true), "a failing local lookup must be left out")
end)

-- A browse list's Search row only searches that list; this one asks every library at once.
test("searching all libraries sends a query with no library chosen", function()
    local plugin = H.newPlugin()
    plugin:searchEverything()
    local dialog = H.shown[#H.shown]
    dialog.input_text = "dune"
    H.findButton(dialog.buttons, "Search").callback()
    local browse
    for _, request in ipairs(H.requests) do
        if request.url:find("/browse?", 1, true) then browse = request.url end
    end
    assert(browse, "no browse request was made")
    assert(browse:find("q=dune", 1, true), "the query was not sent: " .. browse)
    assert(not browse:find("library=", 1, true), "a library was chosen, so it is not a search of everything: " .. browse)
end)

-- A bookmark is a page, not a text range: KOReader keeps it in the same annotation list without a drawer, so it used
-- to be dropped on the way out and invisible everywhere else.
test("a bookmark made here is uploaded as a bookmark", function()
    local plugin, ui = H.newPlugin()
    ui.annotation.annotations = {{page = 7, chapter = "Chapter", text = "in Chapter"}}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 1, applied = {}} end
        return 200, H.encode{revision = 2, applied = {{id = "koreader:bm", revision = 2}}}
    end
    plugin:syncAnnotations(false)
    local sent
    for _, request in ipairs(H.requests) do if request.method == "PUT" then sent = H.decode(request.body).changes[1].annotation end end
    assert(sent, "the bookmark never left the device")
    eq(sent.kind, "bookmark", "a page bookmark must be sent as a bookmark")
    eq(sent.quote, "", "a bookmark has no quote")
    eq(H.decode(sent.locators.koreader).page, 7, "the bookmark's page was not sent")
end)

test("a bookmark made on another reader appears on this one", function()
    local plugin, ui = H.newPlugin()
    local native = H.encode{page = "/body/DocFragment[7]/p[1]"}
    H.respond = function(req)
        if req.method == "GET" then
            return 200, H.encode{revision = 1, applied = {{id = "koreader:bm", revision = 1, kind = "bookmark", locators = {koreader = native}}}}
        end
        return 200, H.encode{revision = 1, applied = {}}
    end
    plugin:syncAnnotations(false)
    eq(#ui.annotation.annotations, 1, "the remote bookmark was not added")
    eq(ui.annotation.annotations[1].page, 7, "the bookmark was added on the wrong page")
    eq(ui.annotation.annotations[1].mochil_id, "koreader:bm", "the imported bookmark was not tagged, so it would be uploaded again")
end)

test("a bookmark deleted elsewhere is removed here", function()
    local plugin, ui = H.newPlugin()
    ui.annotation.annotations = {{page = 7, chapter = "Chapter", text = "in Chapter"}}
    H.respond = function(req)
        if req.method == "GET" then
            return 200, H.encode{revision = 2, applied = {{id = "koreader:bm", deleted = true, revision = 2, kind = "bookmark",
                locators = {koreader = H.encode{page = 7}}}}}
        end
        return 200, H.encode{revision = 2, applied = {}}
    end
    plugin:syncAnnotations(true)
    eq(#ui.annotation.annotations, 0, "the deleted bookmark was left on the page")
end)

-- The direction matters: toggling a page that is already bookmarked deletes it, so an unchanged bookmark must never
-- be toggled at all.
test("a bookmark that is already on the page is left alone", function()
    local plugin, ui = H.newPlugin()
    local items = {{page = 7, chapter = "Chapter", text = "in Chapter", mochil_id = "koreader:bm", mochil_revision = 1}}
    ui.annotation.annotations = items
    H.respond = function(req)
        if req.method == "GET" then
            return 200, H.encode{revision = 1, applied = {{id = "koreader:bm", revision = 1, kind = "bookmark", locators = {koreader = H.encode{page = 7}}}}}
        end
        return 200, H.encode{revision = 1, applied = {}}
    end
    plugin:syncAnnotations(false)
    eq(ui.bookmark.toggles, 0, "an unchanged bookmark was toggled, which deletes it")
    eq(#ui.annotation.annotations, 1, "the bookmark disappeared")
end)

-- A deletion has to say which page it removes, or the other device has nothing to un-bookmark: the page IS the
-- bookmark's identity, and the tombstone used to be sent as a plain highlight with no locator at all.
test("deleting a bookmark tells the other device which page it was on", function()
    local plugin, ui, settings = H.newPlugin()
    ui.annotation.annotations = {}
    local before = plugin:localAnnotations()
    settings.mochil_annotation_snapshot = {}
    -- One saved bookmark, then the reader removes it and a sync notices.
    ui.annotation.annotations = {{page = 7, chapter = "Chapter", text = "in Chapter"}}
    local id, bookmark = next(plugin:localAnnotations())
    settings.mochil_annotation_snapshot = {[id] = {revision = 3, signature = bookmark._signature, kind = "bookmark", locators = bookmark.locators}}
    ui.annotation.annotations = {}
    H.respond = function(req)
        if req.method == "GET" then return 200, H.encode{revision = 4, applied = {}} end
        return 200, H.encode{revision = 5, applied = {{id = id, deleted = true}}}
    end
    -- Two syncs: the first notices the gap, the second sends the deletion.
    plugin:syncAnnotations(false)
    plugin:syncAnnotations(false)
    local tombstone
    for _, request in ipairs(H.requests) do
        if request.method == "PUT" then
            for _, change in ipairs(H.decode(request.body).changes) do
                if change.annotation.deleted then tombstone = change.annotation end
            end
        end
    end
    assert(tombstone, "no deletion was sent")
    eq(tombstone.kind, "bookmark", "the deletion lost the bookmark kind")
    assert(tombstone.locators and H.decode(tombstone.locators.koreader).page == 7, "the deletion did not say which page")
end)

io.write(string.format("%d passed, %d failed\n", passed, #failures))
for _, failure in ipairs(failures) do io.write("FAIL: " .. failure .. "\n") end
os.exit(#failures == 0 and 0 or 1)
