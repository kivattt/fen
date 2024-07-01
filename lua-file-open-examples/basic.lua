-- This script opens a new editor consecutively for every selected file

local editor = os.getenv("EDITOR")
if fen.RuntimeOS == "windows" then
	editor = "notepad"
end

if editor == nil or editor == "" then
	print("Could not find text editor, quitting file open script")
	os.exit(1)
end

local function onlyHasSpaceCharacters(s)
	for i = 1, #s do
		if s:sub(i,i) ~= " " then
			return false
		end
	end

	return true
end

if onlyHasSpaceCharacters(editor) then
	print("Environment variable EDITOR only has spaces in it, quitting file open script")
	os.exit(2)
end

for i = 1, #fen.SelectedFiles do
	os.execute(editor.." "..fen.SelectedFiles[i])
end
