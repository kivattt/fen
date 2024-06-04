local y = 0
for line in io.lines(fen.SelectedFile) do
	-- "[::d]" is a tview style tag that dims the text
	-- https://pkg.go.dev/github.com/rivo/tview#hdr-Styles__Colors__and_Hyperlinks
	fen:PrintSimple("[::d]"..fen:TranslateANSI(fen:Escape(line)), 0, y)

	y = y + 1
	if y >= fen.Height then
		break
	end
end
