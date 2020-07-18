package handlers

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"

	"github.com/f4nt0md3v/tic-tac-go-beeline/app/models"
	"github.com/f4nt0md3v/tic-tac-go-beeline/app/repositories"
)

var (
	upg = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
)

const (
	CmdGenerateNewGame = "GENERATE_NEW_GAME"
	CmdJoinGame        = "JOIN_GAME"
	CmdMakeMove        = "MAKE_MOVE"
)

func WebsocketHandler(c echo.Context) error {
	// Upgrade HTTP connection to WebSocket
	ws, err := upg.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := ws.Close(); err != nil {
			c.Logger().Error(err)
		}
	}()

	c.Logger().Print("Client connected...")

	for {
		var req models.Request
		// Receive a message using the codec
		if err := ws.ReadJSON(&req); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Printf("Received message: %s", req)

		switch req.Command {
		case CmdGenerateNewGame:
			c.Logger().Print("Generating a new game...")
			gameInfo, err := GenerateNewGame(c)
			if err != nil {
				errResp := models.ErrorResponse{
					Code:  http.StatusInternalServerError,
					Error: err.Error(),
				}
				if err := ws.WriteJSON(errResp); err != nil {
					c.Logger().Error(err)
				}
			}

			resp := models.Response{
				Command:  CmdGenerateNewGame,
				GameInfo: *gameInfo,
				Message:  http.StatusText(http.StatusCreated),
			}
			if err := ws.WriteJSON(resp); err != nil {
				c.Logger().Error(err)
			}

		case CmdJoinGame:
			c.Logger().Print("Joining the game game...")
			if req.GameInfo.GameId != "" {
				errResp := models.ErrorResponse{
					Code:  http.StatusBadRequest,
					Error: "No game id provided",
				}
				if err := ws.WriteJSON(errResp); err != nil {
					c.Logger().Error(err)
				}
			}

			gameInfo, err := JoinGame(req.GameInfo.GameId, c)
			if err != nil {
				errResp := models.ErrorResponse{
					Code:  http.StatusInternalServerError,
					Error: err.Error(),
				}
				if err := ws.WriteJSON(errResp); err != nil {
					c.Logger().Error(err)
				}
			}

			resp := models.Response{
				Command:  CmdJoinGame,
				Code:     http.StatusOK,
				GameInfo: *gameInfo,
				Message:  http.StatusText(http.StatusOK),
			}
			if err := ws.WriteJSON(resp); err != nil {
				c.Logger().Error(err)
			}

		case CmdMakeMove:
			c.Logger().Print("Making a move...")
			if req.GameInfo.GameId != "" && req.GameInfo.State != "" {
				errResp := models.ErrorResponse{
					Code:  http.StatusBadRequest,
					Error: "No game state provided",
				}
				if err := ws.WriteJSON(errResp); err != nil {
					c.Logger().Error(err)
				}
			}

			gameInfo, err := MakeMove(req.GameInfo, c)
			if err != nil {
				errResp := models.ErrorResponse{
					Code:  http.StatusInternalServerError,
					Error: err.Error(),
				}
				if err := ws.WriteJSON(errResp); err != nil {
					c.Logger().Error(err)
				}
			}

			resp := models.Response{
				Command:  CmdJoinGame,
				Code:     http.StatusOK,
				GameInfo: *gameInfo,
				Message:  http.StatusText(http.StatusOK),
			}
			if err := ws.WriteJSON(resp); err != nil {
				c.Logger().Error(err)
			}
		default:
			// Send a OK response by default meaning
			// that connection established and the message format is correct
			resp := models.Response{
				Message: http.StatusText(http.StatusOK),
				Code:    http.StatusOK,
			}
			if err := ws.WriteJSON(resp); err != nil {
				c.Logger().Error(err)
			}
		}
	}
}

func GenerateNewGame(c echo.Context) (*models.Game, error) {
	// Generate user id and game id
	userId := uuid.NewV4().String()
	c.Logger().Printf("user_id: %s\n", userId)
	gameId := uuid.NewV4().String()
	c.Logger().Printf("game_id: %s\n", gameId)

	repo := c.Get("GAME_REPO").(*repositories.GameRepo)
	g, err := repo.Create(gameId, userId)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func JoinGame(gameId string, c echo.Context) (*models.Game, error) {
	repo := c.Get("GAME_REPO").(*repositories.GameRepo)
	curGame, err := repo.FindByGameID(gameId)
	if err != nil {
		return nil, err
	}

	// Generate new user id
	userId := uuid.NewV4().String()
	c.Logger().Printf("user_id: %s\n", userId)

	// TODO: this fixes the bug with attempt to connect to ongoing game but for now keep it
	// if curGame.SecondUserId != "" {
	// 	err := errors.New("can't join game")
	// 	return nil, err
	// }

	// Register new user as second user
	curGame.SecondUserId = userId
	// Update game with new user
	err = repo.Update(curGame)
	if err != nil {
		return nil, err
	}

	return curGame, nil
}

func MakeMove(game models.Game, c echo.Context) (*models.Game, error) {
	repo := c.Get("GAME_REPO").(*repositories.GameRepo)
	curGame, err := repo.FindByGameID(game.GameId)
	if err != nil {
		return nil, err
	}
	curGame.State = game.State
	curGame.LastMoveUserId = game.LastMoveUserId
	// Update game with new state
	err = repo.Update(curGame)
	if err != nil {
		return nil, err
	}

	return curGame, nil
}
