--[[
-- File preview script for *.toml files
--]]

-- Only works with ASCII range characters, UTF-8 stuff like æøå will not be shown correctly unfortunately
-- It's written pretty naively, some strings might be colored incorrectly

-- Customize the colors here:
local colorForHeaders, attributesForHeaders = "white", "b" -- Bold
local colorForKeys = "aqua" -- Everything else
local colorForSymbols = "red"
local colorForStrings = "yellow"
local colorForComments = "blue"

local function isSymbol(char)
	local symbols = "{}[],="
	for i = 1, #symbols do
		if char == symbols:sub(i,i) then
			return true
		end
	end
	return false
end

local function isControlChar(char)
	return string.byte(char) < 0x20 or string.byte(char) == 0x7f
end

-- https://toml.io/en/v1.0.0#comment
local function isLegalCommentChar(char)
	if not isControlChar(char) then
		return true
	end

	return char == '\t' or not (string.byte(char) <= 0x08 or (string.byte(char) >= 0x0a and string.byte(char) <= 0x1f) or char == 0x7f)
end

local function styleString(foreground, background, attributes)
	return "["..foreground..":"..background..":"..attributes.."]"
end

local function trimLeftSpaces(s)
	return s:match'^%s*(.*)'
end

local inString, comment = false, false
local invalidFile, invalidFileReason = false, ""
local char, lastChar = '', ''

local y = 0
for line in io.lines(fen.selectedFile) do
	comment = false
	inString = false

	local lineTrimLeftSpaces = trimLeftSpaces(line)
	if lineTrimLeftSpaces:sub(1,1) == '[' then
		fen:PrintSimple(styleString(colorForHeaders, "", attributesForHeaders)..fen:Escape(line), 0, y)
		y = y + 1
		if y >= fen.Height then
			break
		end
		goto continueLine
	end

	for i = 1, #line do
		char = line:sub(i,i)

		local styleForeground = colorForKeys

		if not inString and not comment then
			if char == '#' then
				comment = true
			end
		end

		if comment then
			if not isLegalCommentChar(char) then
				invalidFile = true
				invalidFileReason = "Illegal comment character ("..string.format("0x%x", string.byte(char))..") on line "..y+1
				fen:Print("[::r]"..string.format("0x%x", string.byte(char)).."[red::bR] <- HERE", i-1, y, fen.Width, 0, 0)
				break
			end
			styleForeground = colorForComments
			fen:PrintSimple(styleString(styleForeground, "", "")..char, i-1, y)
			goto continue
		end

		if not inString and char == '"' then
			inString = true
		elseif inString and char == '"' and lastChar ~= '\\' then
			inString = false
			fen:PrintSimple(styleString(colorForStrings, "", "")..char, i-1, y)
			goto continue
		end

		if inString then
			styleForeground = colorForStrings
		end

		if not comment and not inString then
			if isSymbol(char) then
				styleForeground = colorForSymbols
			end
		end

		fen:PrintSimple(styleString(styleForeground, "", "")..char, i-1, y)
		lastChar = char
		::continue::
	end

	if invalidFile then
		fen:Print("[red::b]"..invalidFileReason, 0, y+1, fen.Width, 0, 0)
		fen:Print("[red::b]toml file preview end", 0, y+2, fen.Width, 0, 0)
		break
	end

	y = y + 1
	if y >= fen.Height then
		break
	end
	::continueLine::
end
