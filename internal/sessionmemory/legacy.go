package sessionmemory

import (
	"fmt"
	"strings"
)

func (s *Store) backfillLegacySessions() (int, error) {
	rows, err := s.db.Query(`SELECT s.id
		FROM codex_sessions s
		LEFT JOIN codex_session_capsules c ON c.session_id=s.id
		WHERE c.session_id IS NULL
			OR s.title LIKE '<recommended_plugins>%'
			OR s.title LIKE '<!-- pallium:agents:begin -->%'
			OR s.title LIKE '# AGENTS.md instructions%'
			OR length(s.first_user_message)>10000
			OR s.status='skipped_large_rollout'
		ORDER BY COALESCE(NULLIF(s.updated_at,''),s.created_at)`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	backfilled := 0
	for _, id := range ids {
		sess, err := s.loadSession(id)
		if err != nil {
			return backfilled, err
		}
		messages, err := s.loadSessionMessages(id)
		if err != nil {
			return backfilled, err
		}
		messagesSeen := len(messages)
		cleanMessages := make([]Message, 0, len(messages))
		firstRealUserMessage := ""
		for _, message := range messages {
			if message.Role == "user" {
				message.Text = capMessageText(normalizeUserText(message.Text))
				if message.Text == "" {
					continue
				}
				if firstRealUserMessage == "" {
					firstRealUserMessage = short(message.Text, maxStoredFirstUserText)
				}
			}
			cleanMessages = append(cleanMessages, message)
		}
		messages = cleanMessages
		sess.FirstUserMessage = short(first(normalizeUserText(sess.FirstUserMessage), firstRealUserMessage), maxStoredFirstUserText)
		sess.Title = short(first(normalizeUserText(sess.Title), sess.FirstUserMessage, firstRealUserMessage, "Session "+short(sess.ID, 12)), 240)
		sess.LastAgentMessage = capMessageText(sess.LastAgentMessage)
		coverage := SessionCoverage{
			Mode:            "legacy",
			MessagesSeen:    messagesSeen,
			MessagesStored:  len(messages),
			MessagesDropped: messagesSeen - len(messages),
			Warning:         "Continuity was rebuilt from previously indexed messages because the original source transcript was unavailable during sync.",
		}
		if len(messages) > largeTranscriptHeadMessages+largeTranscriptTailMessages {
			coverage.Mode = "sampled"
			coverage.MessagesDropped = len(messages) - largeTranscriptHeadMessages - largeTranscriptTailMessages
			messages = append(append([]Message{}, messages[:largeTranscriptHeadMessages]...), messages[len(messages)-largeTranscriptTailMessages:]...)
			coverage.MessagesStored = len(messages)
			coverage.Warning = fmt.Sprintf("Legacy continuity sampled the first %d and last %d of %d previously indexed messages because the original source transcript was unavailable.", largeTranscriptHeadMessages, largeTranscriptTailMessages, coverage.MessagesSeen)
		}
		if sess.Status == "skipped_large_rollout" {
			sess.Status = "legacy_recovered"
			var retainedErrors []string
			for _, message := range sess.Errors {
				if !strings.Contains(message, "skipped full parse") {
					retainedErrors = append(retainedErrors, message)
				}
			}
			sess.Errors = retainedErrors
		}
		parsed := ParsedSession{
			Session:     sess,
			Messages:    messages,
			EventCounts: map[string]int{"legacy_backfill": 1},
			Coverage:    coverage,
			SearchBlob: truncate(strings.Join([]string{
				sess.Title,
				sess.CWD,
				strings.Join(sess.Commands, "\n"),
				strings.Join(sess.FilesTouched, "\n"),
				messagesText(messages),
			}, "\n"), maxSearchBlobText),
		}
		if err := s.upsert(parsed, map[string]any{"legacy_backfill": true}); err != nil {
			return backfilled, err
		}
		backfilled++
	}
	return backfilled, nil
}

func (s *Store) loadSessionMessages(id string) ([]Message, error) {
	rows, err := s.db.Query(`SELECT line_no,COALESCE(timestamp,''),COALESCE(role,''),COALESCE(kind,''),COALESCE(text,'') FROM codex_session_messages WHERE session_id=? ORDER BY line_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.LineNo, &message.Timestamp, &message.Role, &message.Kind, &message.Text); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}
