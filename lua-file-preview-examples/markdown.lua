--[[
-- FIXME: Only output style tags when the style changes
--]]

local function trimLeftSpaces(s)
	return s:match'^%s*(.*)'
end

local function trimLeftHashes(s)
	return s:match'^#*(.*)'
end

local function isEmphasis(char)
	return char == '*' or char == '_'
end

-- https://spec.commonmark.org/0.31.2/#code-fence
local function isCodeFence(char)
	return char == '`' or char == '~'
end

local italic, bold, codeblock, backtickString = false, false, false, false
local char, lastChar = '', ''
local style = ""

local y = 0
for line in io.lines(fen.SelectedFile) do
	local lineTrimLeftSpaces = trimLeftSpaces(line)
	local xOffset = 0

	local codeFenceChars = 0
	for j = 1, #lineTrimLeftSpaces do
		local c = lineTrimLeftSpaces:sub(j,j)
		if not isCodeFence(c) then
			break
		end
		codeFenceChars = codeFenceChars + 1
	end
	if codeFenceChars == 3 then
		codeblock = not codeblock
		goto continueLine
	end

	local headingDepth = 0
	if not codeblock then
		for j = 1, #lineTrimLeftSpaces do
			local c = lineTrimLeftSpaces:sub(j,j)
			if c ~= '#' then
				if c ~= ' ' then
					headingDepth = 0
				end
				break
			end
			headingDepth = headingDepth + 1
			if headingDepth > 6 then
				headingDepth = 0
				break
			end
		end

		if headingDepth > 0 then
			-- :skull:
			line = trimLeftSpaces(trimLeftHashes(trimLeftSpaces(line)))
		end
	end

	for i = 1, #line do
		char = line:sub(i,i)

		if not codeblock then
			if char == '`' and lastChar ~= '`' then
				backtickString = not backtickString
				-- It messes up table alignment if we skip over the backticks, so we just replace them with blank space instead
				--xOffset = xOffset - 1
				fen:PrintSimple("[:black] ", i+xOffset-1, y)
				lastChar = char
				goto continue
			end

			if not backtickString then
				if isEmphasis(lastChar) and isEmphasis(char) then
					bold = not bold
					italic = false
					xOffset = xOffset - 1
					lastChar = char
					goto continue
				elseif isEmphasis(char) then
					italic = not italic
					xOffset = xOffset - 1
					lastChar = char
					goto continue
				end
			end
		end

		local styleForeground, styleBackground, styleAttributes = "", "", ""
		if bold then
			styleAttributes = styleAttributes.."b" -- bold
		end
		if italic then
			styleAttributes = styleAttributes.."i" -- italic
		end
		if headingDepth > 0 then
			styleAttributes = styleAttributes.."b" -- bold
			styleForeground = "white"
		end
		if codeblock then
			styleForeground = fen:ColorToString(fen:NewRGBColor(0, 255, 0))
			styleBackground = "black"
		end
		if backtickString then
			styleAttributes = styleAttributes.."d" -- dim
			styleBackground = "black"
		end

		if not (styleForeground == "" and styleAttributes == "" and styleBackground == "") then
			style = "["..styleForeground..":"..styleBackground..":"..styleAttributes.."]"
		else
			style = ""
		end

		fen:PrintSimple(style..char, i+xOffset-1, y)
		lastChar = char
	    ::continue::
	end

	if lineTrimLeftSpaces:sub(1,1) == "-" then
		fen:PrintSimple("[::d]●", 0, y)
	end

	y = y + 1
	if y >= fen.Height then
		break
	end
	::continueLine::
end
