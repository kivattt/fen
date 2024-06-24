--[[
-- Works for fen v1.2.3 or above
--]]

-- Only works with ASCII range characters, UTF-8 stuff like æøå will not be shown correctly unfortunately
-- It's written pretty naively, some strings might be colored incorrectly

local colorForStrings = "yellow"
local colorForComments = "blue"
local singleQuotesAreStrings = true
local doubleForwardSlashesAreComments = false

local function isComment(char, nextChar)
	return char == '#' or (doubleForwardSlashesAreComments and (char == '/' and nextChar == '/'))
end

local function isString(char, lastStringChar)
	if lastStringChar == '' then
		return char == '"' or (singleQuotesAreStrings and char == '\'')
	end

	return char == lastStringChar
end

local function styleString(foreground, background, attributes)
	return "["..foreground..":"..background..":"..attributes.."]"
end

local inString, comment = false, false
local char, nextChar, lastChar, lastLastChar, lastStringChar = '', '', '', '', ''

local y = 0
for line in io.lines(fen.selectedFile) do
	comment = false
	inString = false
	local x = 0

	for i = 1, #line do
		char = line:sub(i,i)
		local nextIdx = math.min(#line, i + 1)
		nextChar = line:sub(nextIdx,nextIdx)

		if not inString then
			if not comment and isComment(char, nextChar) then
				comment = true
			end
		end

		if not comment then
			if not inString and isString(char, lastStringChar) then
				inString = true
				lastStringChar = char
			elseif inString and isString(char, lastStringChar) and (lastChar ~= '\\' or (lastChar == '\\' and lastLastChar == '\\')) then
				inString = false
				lastStringChar = ''

				x = x + fen:PrintSimple(styleString(colorForStrings, "", "")..char, x, y)
				goto continue
			end
		end

		local foregroundColor = ""
		if comment then
			foregroundColor = colorForComments
		elseif inString then
			foregroundColor = colorForStrings
		end

		x = x + fen:PrintSimple(styleString(foregroundColor, "", "")..char, x, y)

		lastLastChar = lastChar
		lastChar = char
		::continue::
	end

	y = y + 1
	if y >= fen.Height then
		break
	end
end
