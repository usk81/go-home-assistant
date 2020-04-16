package assistant

import (
	embedded "google.golang.org/genproto/googleapis/assistant/embedded/v1alpha2"
)

// GoogleSampleRate is sample rate for google home speaker
const GoogleSampleRate int32 = 16000

var (
	// keep the state in memory to advance the conversation.
	conversationState []byte
)

func GetDefaultConfig() embedded.AssistConfig {
	return embedded.AssistConfig{
		AudioOutConfig:  getAudioOutConfig(),
		DebugConfig:     getDebugConfig(),
		DeviceConfig:    getDeviceConfig(),
		DialogStateIn:   getDialogStateIn(),
		ScreenOutConfig: getScreenOutConfig(),
	}
}

func getAudioInConfig() *embedded.AudioInConfig {
	return &embedded.AudioInConfig{
		Encoding:        embedded.AudioInConfig_FLAC,
		SampleRateHertz: GoogleSampleRate,
	}
}

func getAudioOutConfig() *embedded.AudioOutConfig {
	return &embedded.AudioOutConfig{
		Encoding:         embedded.AudioOutConfig_LINEAR16,
		SampleRateHertz:  GoogleSampleRate,
		VolumePercentage: 60,
	}
}

func getDebugConfig() *embedded.DebugConfig {
	return &embedded.DebugConfig{
		ReturnDebugInfo: true,
	}
}

func getDeviceConfig() *embedded.DeviceConfig {
	return &embedded.DeviceConfig{
		DeviceId:      "go-home-assistant",
		DeviceModelId: "go-home-assistant-20200401_macos",
	}
}

func getDialogStateIn() *embedded.DialogStateIn {
	return &embedded.DialogStateIn{
		ConversationState: conversationState,
		LanguageCode:      "en-US",
		IsNewConversation: false,
	}
}

func getScreenOutConfig() *embedded.ScreenOutConfig {
	return &embedded.ScreenOutConfig{
		ScreenMode: embedded.ScreenOutConfig_OFF,
	}
}
