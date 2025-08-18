package model

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyResultMsg struct {
	key       *rsa.PrivateKey
	publicKey []byte
	isNewUser bool
	err       error
}

type Styles struct {
	BorderColor    lipgloss.Color
	InputField     lipgloss.Style
	Button         lipgloss.Style
	SelectedButton lipgloss.Style
	// Tabs
	ActiveTabBorder lipgloss.Border
	ActiveTab       lipgloss.Style
	InactiveTab     lipgloss.Style
	TabGap          lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.BorderColor = lipgloss.Color("69")

	s.InputField = lipgloss.NewStyle().
		BorderForeground(s.BorderColor).
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(20)

	s.Button = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("57")).
		Padding(0, 3).
		MarginTop(1).
		MarginRight(1)

	s.SelectedButton = s.Button.
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("105")).
		Underline(true)

	// --- Tab Styles ---
	inactiveTabBorder := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}
	activeTabBorder := inactiveTabBorder
	activeTabBorder.Bottom = " "

	s.ActiveTab = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 3).
		Foreground(lipgloss.Color("205")).
		Border(activeTabBorder, true).
		BorderForeground(s.BorderColor)

	s.InactiveTab = lipgloss.NewStyle().
		Padding(0, 3).
		Foreground(lipgloss.Color("240")).
		Border(inactiveTabBorder, true).
		BorderForeground(s.BorderColor)

	// Gap style to put between tabs
	s.TabGap = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(s.BorderColor).
		PaddingRight(1)

	return s
}

type authState int

const (
	stateReadyForInput = iota
	stateBusy
)

// --- AuthModel (MetaModel) ---
type AuthModel struct {
	styles        *Styles
	usernameInput textinput.Model
	state         authState
	statusMessage string
	currentKey    *rsa.PrivateKey
	conn          *net.TCPConn
	width, height int
}

func NewAuthModel(conn *net.TCPConn) AuthModel {
	styles := DefaultStyles()

	tiUser := textinput.New()
	tiUser.Focus()
	tiUser.Placeholder = "Username"
	tiUser.CharLimit = 32
	tiUser.Width = 20

	return AuthModel{
		styles:        styles,
		usernameInput: tiUser,
		state:         stateReadyForInput,
		statusMessage: "Enter your username to login or register",
		conn:          conn,
	}
}

func (m *AuthModel) SetConn(conn *net.TCPConn) {
	m.conn = conn
}

func (m *AuthModel) SetDimension(height, width int) {
	m.width = width
	m.height = height
}

func (m *AuthModel) logKeyForDebug() {
	if m.currentKey == nil {
		log.Println("DEBUG: m.currentKey is nil")
		return
	}

	// Marshal the key into a DER-encoded format
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(m.currentKey)
	if err != nil {
		log.Printf("DEBUG: Failed to marshal key for logging: %v", err)
		return
	}

	// Wrap the DER data in a PEM block
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	}

	// Encode to memory and print
	log.Printf("DEBUG: Current Key:\n%s", string(pem.EncodeToMemory(privateKeyPEM)))
}

func (m AuthModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateReadyForInput {
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit

			case tea.KeyEnter:
				m.state = stateBusy
				m.statusMessage = "Connecting..."
				username := m.usernameInput.Value()

				return m, func() tea.Msg {
					// Try to load the key for the user
					key, err := loadKey(username)
					if err != nil {
						// KEY NOT FOUND
						// Generate a new key pair
						privKey, pubKeyBytes, genErr := generateAndSaveKeys(username)
						return keyResultMsg{
							key:       privKey,
							publicKey: *pubKeyBytes,
							isNewUser: true,
							err:       genErr,
						}
					} else {
						// KEY FOUND
						return keyResultMsg{
							key:       key,
							isNewUser: false,
							err:       nil,
						}
					}
				}
			default:
				m.usernameInput, cmd = m.usernameInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
  case keyResultMsg:
		if msg.err != nil {
			m.state = stateReadyForInput
			m.statusMessage = "Error handling key: " + msg.err.Error()
			return m, nil
		}

		// Key was loaded/generated successfully, store it on the model.
		m.currentKey = msg.key
		username := m.usernameInput.Value()

		if msg.isNewUser {
			m.statusMessage = "New user detected. Registering..."
			communication.SendRegisterRequestPacket(m.conn, username, msg.publicKey)
		} else {
			m.statusMessage = "Existing user. Requesting challenge..."
			m.logKeyForDebug()
			communication.SendLoginChallengeRequestPacket(m.conn, username)
		}
		return m, nil

	// --- Handle Messages From Server ---
	case communication.RegistrationResultMsg:
		if !msg.Success {
			m.state = stateReadyForInput
			m.statusMessage = "Registration failed: " + msg.Message
			return m, nil
		}
		// Registration was successful, server sent back a challenge
		m.statusMessage = "Registration successful! Logging in..."
		signedChallenge, err := signChallenge(m.currentKey, msg.Challenge)
		if err != nil {
			m.state = stateReadyForInput
			m.statusMessage = "Failed to sign challenge: " + err.Error()
			return m, nil
		}
		username := m.usernameInput.Value()
		communication.SendAuthRequestPacket(m.conn, username, signedChallenge)
		return m, nil

	case communication.ChallengeReceivedMsg:
		m.statusMessage = "Challenge received. Signing..."
		if m.currentKey == nil {
			log.Print("Error m.currentkey is nil")
		} else {
			log.Printf("This should not be called")
		}
		m.logKeyForDebug()
		signedChallenge, err := signChallenge(m.currentKey, msg.Challenge)
		if err != nil {
			m.state = stateReadyForInput
			m.statusMessage = "Failed to sign challenge: " + err.Error()
			return m, nil
		}
		username := m.usernameInput.Value()
		communication.SendAuthRequestPacket(m.conn, username, signedChallenge)
		return m, nil

	case communication.AuthResultMsg:
		if !msg.Success {
			m.state = stateReadyForInput
			m.statusMessage = "Login failed: " + msg.Message
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)

}

func (m AuthModel) View() string {
	informationMessage := "Press 'Enter' to send the username.\nPress 'Escape' to exit"
	inputUserStyle := m.styles.InputField
	inputUserStyle = inputUserStyle.BorderForeground(lipgloss.Color("205"))
	renderedInput := inputUserStyle.Render(m.usernameInput.View())
	content := lipgloss.JoinVertical(lipgloss.Left,
		informationMessage,
		renderedInput,
		m.statusMessage,
	)
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

}

func loadKey(username string) (*rsa.PrivateKey, error) {
	keyFile, err := getKeyPath(username)
	if err != nil {
		return nil, err
	}
	pemData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing private Key")
	}
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("key is not an RSA private key")
	}
	return privateKey, nil
}

func generateAndSaveKeys(username string) (*rsa.PrivateKey, *[]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	privateKeyDer, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	privateKeyPem := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDer,
	}

	keyFile, err := getKeyPath(username)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Create(keyFile)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	if err := pem.Encode(file, privateKeyPem); err != nil {
		return nil, nil, err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, &publicKeyBytes, nil
}

func signChallenge(key *rsa.PrivateKey, challenge []byte) ([]byte, error) {
	if key == nil {
		log.Print("Error key is nil")
	}
	hashed := sha256.Sum256(challenge)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func getKeyPath(username string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dirPath := filepath.Join(homeDir, ".config", "term_arena", "keys")
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dirPath, fmt.Sprintf("%s.key", username)), nil
}
