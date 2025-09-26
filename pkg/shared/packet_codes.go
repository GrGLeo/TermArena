package shared

// Packet code ranges (0-255 total)
// Each service gets a dedicated range to prevent conflicts

const (
	// Authentication packets (0-49)
	CodeRegisterRequest        = 0
	CodeRegisterResponse       = 1
	CodeLoginChallengeRequest  = 2
	CodeLoginChallengeResponse = 3
	CodeAuthRequest            = 4
	CodeAuthResponse           = 5

	// Room management packets (50-99)
	CodeRoomRequest     = 50
	CodeRoomCreate      = 51
	CodeRoomJoin        = 52
	CodeLookRoom        = 53
	CodeMoveToLobby     = 54
	CodeUpdateSpellReq  = 55
	CodeUpdateSpellRes  = 56
	CodeGameServerReady = 57

	// Game packets (100-149)
	CodeGameStart      = 100
	CodeAction         = 101
	CodeBoard          = 102
	CodeDelta          = 103
	CodeGameClose      = 104
	CodeEndGame        = 105
	CodeSpellSelection = 106
	CodeShopRequest    = 107
	CodeShopResponse   = 108
	CodePurchaseItem   = 109

	// Message packets (150-199)
	CodeMessage         = 150
	CodeMessageResponse = 151
	CodeMessageError    = 152

	// Special packets (250-255)
	CodeRateLimit = 255
)
