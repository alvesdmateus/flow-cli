-- flow.nvim - AI coding assistant for Neovim
-- Uses flow-cli's LSP server for AI-powered coding assistance

local M = {}

-- Default configuration
M.config = {
    server_path = "flow",
    model = "",
    provider = "ollama",
    auto_start = true,
    log_level = "info",
    filetypes = {
        "go", "python", "javascript", "typescript", "rust",
        "java", "c", "cpp", "cs", "ruby", "php", "swift", "kotlin"
    },
    keymaps = {
        explain = "<leader>ve",
        tests = "<leader>vt",
        refactor = "<leader>vr",
        fix = "<leader>vf",
        docs = "<leader>vd",
        chat = "<leader>vc",
        prompt = "<leader>vp",
    },
}

-- State
local client_id = nil
local output_buf = nil
local output_win = nil

-- Setup function
function M.setup(opts)
    M.config = vim.tbl_deep_extend("force", M.config, opts or {})

    -- Setup LSP
    if M.config.auto_start then
        M.start_server()
    end

    -- Setup keymaps
    M.setup_keymaps()

    -- Setup commands
    M.setup_commands()

    -- Setup user commands
    vim.api.nvim_create_user_command("FlowStart", M.start_server, {})
    vim.api.nvim_create_user_command("FlowStop", M.stop_server, {})
    vim.api.nvim_create_user_command("FlowExplain", M.explain_code, { range = true })
    vim.api.nvim_create_user_command("FlowTests", M.generate_tests, { range = true })
    vim.api.nvim_create_user_command("FlowRefactor", M.refactor_code, { range = true })
    vim.api.nvim_create_user_command("FlowFix", M.fix_error, {})
    vim.api.nvim_create_user_command("FlowDocs", M.generate_docs, { range = true })
    vim.api.nvim_create_user_command("FlowChat", M.open_chat, {})
    vim.api.nvim_create_user_command("FlowPrompt", function(args)
        M.run_prompt(args.args)
    end, { nargs = "?" })
end

-- Start the LSP server
function M.start_server()
    if client_id then
        vim.notify("Flow server is already running", vim.log.levels.INFO)
        return
    end

    local cmd = { M.config.server_path, "lsp" }
    if M.config.model ~= "" then
        table.insert(cmd, "--model")
        table.insert(cmd, M.config.model)
    end

    local config = {
        name = "flow",
        cmd = cmd,
        root_dir = vim.fn.getcwd(),
        filetypes = M.config.filetypes,
        settings = {},
        capabilities = vim.lsp.protocol.make_client_capabilities(),
        on_attach = function(client, bufnr)
            M.on_attach(client, bufnr)
        end,
        on_init = function(client)
            vim.notify("Flow language server initialized", vim.log.levels.INFO)
        end,
        on_exit = function(code, signal, client_id)
            vim.notify("Flow language server stopped", vim.log.levels.INFO)
        end,
    }

    client_id = vim.lsp.start_client(config)
    if client_id then
        -- Attach to all matching buffers
        for _, bufnr in ipairs(vim.api.nvim_list_bufs()) do
            local ft = vim.bo[bufnr].filetype
            if vim.tbl_contains(M.config.filetypes, ft) then
                vim.lsp.buf_attach_client(bufnr, client_id)
            end
        end
        vim.notify("Flow language server started", vim.log.levels.INFO)
    else
        vim.notify("Failed to start Flow language server", vim.log.levels.ERROR)
    end
end

-- Stop the LSP server
function M.stop_server()
    if not client_id then
        vim.notify("Flow server is not running", vim.log.levels.INFO)
        return
    end

    vim.lsp.stop_client(client_id)
    client_id = nil
end

-- On attach callback
function M.on_attach(client, bufnr)
    -- Set buffer-local keymaps
    local opts = { buffer = bufnr, noremap = true, silent = true }

    if M.config.keymaps.explain then
        vim.keymap.set("v", M.config.keymaps.explain, M.explain_code, opts)
    end
    if M.config.keymaps.tests then
        vim.keymap.set({ "n", "v" }, M.config.keymaps.tests, M.generate_tests, opts)
    end
    if M.config.keymaps.refactor then
        vim.keymap.set("v", M.config.keymaps.refactor, M.refactor_code, opts)
    end
    if M.config.keymaps.fix then
        vim.keymap.set("n", M.config.keymaps.fix, M.fix_error, opts)
    end
    if M.config.keymaps.docs then
        vim.keymap.set({ "n", "v" }, M.config.keymaps.docs, M.generate_docs, opts)
    end
end

-- Setup global keymaps
function M.setup_keymaps()
    local opts = { noremap = true, silent = true }

    if M.config.keymaps.chat then
        vim.keymap.set("n", M.config.keymaps.chat, M.open_chat, opts)
    end
    if M.config.keymaps.prompt then
        vim.keymap.set("n", M.config.keymaps.prompt, function()
            vim.ui.input({ prompt = "Flow: " }, function(input)
                if input then
                    M.run_prompt(input)
                end
            end)
        end, opts)
    end
end

-- Setup commands
function M.setup_commands()
    -- Auto-attach to new buffers
    vim.api.nvim_create_autocmd("FileType", {
        pattern = M.config.filetypes,
        callback = function(args)
            if client_id then
                vim.lsp.buf_attach_client(args.buf, client_id)
            end
        end,
    })
end

-- Get visual selection
local function get_visual_selection()
    local start_pos = vim.fn.getpos("'<")
    local end_pos = vim.fn.getpos("'>")
    local lines = vim.fn.getline(start_pos[2], end_pos[2])

    if #lines == 0 then
        return ""
    end

    -- Adjust for partial line selection
    lines[#lines] = string.sub(lines[#lines], 1, end_pos[3])
    lines[1] = string.sub(lines[1], start_pos[3])

    return table.concat(lines, "\n")
end

-- Execute a Flow command
local function execute_command(command, args)
    if not client_id then
        vim.notify("Flow server is not running. Use :FlowStart to start it.", vim.log.levels.WARN)
        return
    end

    local params = {
        command = command,
        arguments = args,
    }

    vim.lsp.buf_request(0, "workspace/executeCommand", params, function(err, result)
        if err then
            vim.notify("Flow command failed: " .. vim.inspect(err), vim.log.levels.ERROR)
            return
        end

        if result and result.message then
            M.show_output(result.message)
        end
    end)
end

-- Show output in a floating window
function M.show_output(content)
    -- Create or reuse buffer
    if not output_buf or not vim.api.nvim_buf_is_valid(output_buf) then
        output_buf = vim.api.nvim_create_buf(false, true)
        vim.bo[output_buf].buftype = "nofile"
        vim.bo[output_buf].bufhidden = "wipe"
        vim.bo[output_buf].filetype = "markdown"
    end

    -- Set content
    local lines = vim.split(content, "\n")
    vim.api.nvim_buf_set_lines(output_buf, 0, -1, false, lines)

    -- Calculate window size
    local width = math.min(80, vim.o.columns - 4)
    local height = math.min(#lines + 2, math.floor(vim.o.lines * 0.6))

    -- Create floating window
    local win_opts = {
        relative = "editor",
        width = width,
        height = height,
        col = math.floor((vim.o.columns - width) / 2),
        row = math.floor((vim.o.lines - height) / 2),
        style = "minimal",
        border = "rounded",
        title = " Flow ",
        title_pos = "center",
    }

    -- Close existing window
    if output_win and vim.api.nvim_win_is_valid(output_win) then
        vim.api.nvim_win_close(output_win, true)
    end

    output_win = vim.api.nvim_open_win(output_buf, true, win_opts)

    -- Set window options
    vim.wo[output_win].wrap = true
    vim.wo[output_win].linebreak = true

    -- Close on q or Escape
    vim.keymap.set("n", "q", function()
        vim.api.nvim_win_close(output_win, true)
    end, { buffer = output_buf })
    vim.keymap.set("n", "<Esc>", function()
        vim.api.nvim_win_close(output_win, true)
    end, { buffer = output_buf })
end

-- Command implementations
function M.explain_code()
    local selection = get_visual_selection()
    if selection == "" then
        vim.notify("Please select code to explain", vim.log.levels.WARN)
        return
    end

    execute_command("flow.explainCode", {
        vim.uri_from_bufnr(0),
        { start = vim.fn.getpos("'<"), ["end"] = vim.fn.getpos("'>") },
        selection
    })
end

function M.generate_tests()
    local selection = get_visual_selection()
    local content = selection ~= "" and selection or vim.api.nvim_buf_get_lines(0, 0, -1, false)
    if type(content) == "table" then
        content = table.concat(content, "\n")
    end

    execute_command("flow.generateTests", {
        vim.uri_from_bufnr(0),
        {},
        content
    })
end

function M.refactor_code()
    local selection = get_visual_selection()
    if selection == "" then
        vim.notify("Please select code to refactor", vim.log.levels.WARN)
        return
    end

    execute_command("flow.refactor", {
        vim.uri_from_bufnr(0),
        { start = vim.fn.getpos("'<"), ["end"] = vim.fn.getpos("'>") },
        selection
    })
end

function M.fix_error()
    -- Get diagnostics at cursor
    local diagnostics = vim.diagnostic.get(0, { lnum = vim.fn.line(".") - 1 })
    if #diagnostics == 0 then
        vim.notify("No errors at cursor position", vim.log.levels.INFO)
        return
    end

    local diag = diagnostics[1]
    execute_command("flow.fixError", {
        vim.uri_from_bufnr(0),
        diag
    })
end

function M.generate_docs()
    local selection = get_visual_selection()
    local content = selection ~= "" and selection or vim.api.nvim_buf_get_lines(0, 0, -1, false)
    if type(content) == "table" then
        content = table.concat(content, "\n")
    end

    execute_command("flow.generateDocs", {
        vim.uri_from_bufnr(0),
        {},
        content
    })
end

function M.open_chat()
    -- Simple chat implementation using input/output
    local history = {}

    local function chat_loop()
        vim.ui.input({ prompt = "You: " }, function(input)
            if not input or input == "" then
                return
            end

            if input == "/quit" or input == "/exit" then
                vim.notify("Chat ended", vim.log.levels.INFO)
                return
            end

            table.insert(history, { role = "user", content = input })

            execute_command("flow.runPrompt", { input })
            -- Note: Response will be shown in floating window

            -- Continue chat
            vim.defer_fn(chat_loop, 100)
        end)
    end

    vim.notify("Flow Chat started. Type /quit to exit.", vim.log.levels.INFO)
    chat_loop()
end

function M.run_prompt(prompt)
    if not prompt or prompt == "" then
        vim.ui.input({ prompt = "Flow: " }, function(input)
            if input and input ~= "" then
                execute_command("flow.runPrompt", { input })
            end
        end)
    else
        execute_command("flow.runPrompt", { prompt })
    end
end

return M
