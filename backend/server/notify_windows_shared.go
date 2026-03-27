package server

import (
	"bytes"
	"encoding/xml"
	"strings"
)

func buildWindowsToastScript(title, message string) string {
	titleText := escapeWindowsToastXML(strings.TrimSpace(title))
	messageText := escapeWindowsToastXML(strings.TrimSpace(message))
	toastXML := "<toast><visual><binding template=\"ToastGeneric\"><text>" + titleText + "</text><text>" + messageText + "</text></binding></visual></toast>"

	return strings.Join([]string{
		"[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null",
		"[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null",
		"$appId = 'Microsoft.Windows.Explorer'",
		"$xml = New-Object Windows.Data.Xml.Dom.XmlDocument",
		"$xml.LoadXml('" + escapePowerShellSingleQuotedString(toastXML) + "')",
		"$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)",
		"[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)",
	}, "; ")
}

func escapeWindowsToastXML(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		return value
	}
	return buffer.String()
}

func escapePowerShellSingleQuotedString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func normalizeWindowsToastMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "You have a new notification."
	}

	return trimmed
}
