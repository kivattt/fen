--[[
-- File preview for .gitignore files in Git repositories
--]]

local function isSpecialChar(char)
	return char == '/' or char == '*' or char == '?'
end

-- Customize the colors here:
local commentColor = "[teal]"
local specialCharColor = "[orange]"
local captureColor = "[aqua]"

local y = 0
local inCapture = false
for line in io.lines(fen.SelectedFile) do
	if line:sub(1,1) == '#' then
		fen:PrintSimple(commentColor..line, 0, y)
		goto continue
	end

	for i = 1, #line do
		local style = ""
		local c = line:sub(i,i)

		if not inCapture and c == '[' then
			inCapture = true
		end

		if inCapture then
			style = captureColor
		elseif isSpecialChar(c) then
			style = specialCharColor
		end

		if inCapture and c == ']' then
			inCapture = false
		end

		fen:PrintSimple(style..c, i-1, y)
	end

    ::continue::
	y = y + 1
	if y >= fen.Height then
		break
	end
end
