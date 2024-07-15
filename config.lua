-- All available options, default values
fen.ui_borders = false
fen.no_write = false
fen.print_path_on_open = false
fen.mouse = true
fen.hidden_files = true -- Show hidden files
fen.folders_first = true
fen.terminal_title = true -- Only applies to Linux, sets the terminal title to "fen <version>"
fen.show_hostname = true -- Only applies to Linux, shows username@hostname in the top left
fen.show_help_text = true
fen.sort_by = "none" -- "fen -h" for valid values
fen.sort_reverse = false
fen.file_event_interval_ms = 300 -- How often to update the screen on file events, if set to 0, it updates on every event

-- The fen version string
print(fen.version) -- Something like "v1.3.0"

-- The current operating system
print(fen.runtime_os) -- "linux", "darwin" (for macOS), "freebsd", "windows"

-- The OS-specific config path
print(fen.config_path) -- Something like "/home/YOUR_USER/.config/fen/" (always ends in a slash)

-- You can use fen.runtime_os to let your config have specific behaviour on different operating systems
local textEditor = os.getenv("EDITOR")
if fen.runtime_os == "windows" then
	textEditor = "notepad"
end

-- All keys except "script" are lists, but if you only need 1 element you can omit the curly braces

-- When you press right arrow, L or Enter, you open a file.
-- You can customize what programs to use for different file types
fen.open = {
	{
		program = {textEditor},
		match = {"*"}, -- A "*" matches any file, details: https://pkg.go.dev/path/filepath#Match
		do_not_match = {"*.exe"} -- You can exclude certain file types
	},
	{
		-- You can use Lua scripts to open files
		script = fen.config_path.."open.lua",
		match = {"*"}
	}
}

-- In both fen.open and fen.preview, the program to use is determined by the first match.
-- Entries lower down will only be used if the ones above did not match.
--
-- Values in "program" do not expand tildes like "~/some/file.sh"
-- If you want to use a shell script, it has to have a shebang or you need to explicitly invoke the appropriate shell like "bash /some/file.sh"
fen.preview = {
	{
		-- If the first command exits with a non-zero exit code, the next one in the list will be ran
		program = {"head -n 100", "cat"}, -- You can use ' ' for command-line arguments
		match = {"*"},
		do_not_match = {"*.txt", "*.md"} -- When you've selected a .txt or .md file, the Lua script below will preview it
	},
	{
		-- You can use Lua scripts to preview files
		-- https://github.com/kivattt/fen/blob/main/lua-file-preview-examples/rainbow.lua
		script = fen.config_path.."rainbow.lua",
		match = {"*"},
	}
}
