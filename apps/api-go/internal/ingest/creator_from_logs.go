package ingest

// CreatorFromCreateLogs, bir transaction'ın log mesajlarından pump.fun creator'ını (CreateEvent
// `user` pubkey'i) çıkarır. WS logsSubscribe ve REST getTransaction aynı "Program data:" formatını
// verdiğinden decode mantığı (hasCreateInstruction/programDataAll/parseCreateEvent) birebir reuse edilir.
// Create yoksa ya da tanınmazsa ok=false (dürüst boş).
func CreatorFromCreateLogs(logs []string) (creator string, ok bool) {
	if !hasCreateInstruction(logs) {
		return "", false
	}
	for _, raw := range programDataAll(logs) {
		if ev, ok := parseCreateEvent(raw); ok && ev.creator != "" {
			return ev.creator, true
		}
	}
	return "", false
}
