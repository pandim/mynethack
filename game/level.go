package game

import (
	"log"

)

const (
	TileWall  = 0
	TileFloor = 1
)

const (
	PlayerFOVRadius = 8
)

// 🆕 СТРУКТУРА ЛОВУШКИ
type Trap struct {
	X, Y      int
	Triggered bool // Сработала ли ловушка
}

type Tile struct {
	Type     int
	Visible  bool
	Explored bool
}

type Room struct {
	X, Y, W, H int
}

type point struct {
	X, Y int
}

type Level struct {
	Width    int
	Height   int
	Depth    int
	Tiles    [][]Tile
	Monsters []*Monster
	Items    []*Item
	
	Merchants []*Merchant
	Altars    []*Altar
	Chests    []*Chest
	
	// 🆕 ЛОВУШКИ И СЕКРЕТНЫЕ ЛЕСТНИЦЫ
	Traps                 []*Trap
	SecretStairsDownX     int
	SecretStairsDownY     int
	SecretStairsTargetDepth int // На какой уровень ведет (обычно Depth + 2)
	
	VisitCount int
	
	StairsUp    bool
	StairsDown  bool
	StairsUpX   int
	StairsUpY   int
	StairsDownX int
	StairsDownY int
	
	Rooms  []Room      `json:"Rooms"`
	logger *log.Logger `json:"-"`
}

func NewLevel(width, height int, depth int, logger ...*log.Logger) *Level {
	var lgr *log.Logger
	if len(logger) > 0 { lgr = logger[0] }

	if width < 1 { width = 1 }
	if height < 1 { height = 1 }
	if depth < 1 { depth = 1 }

	level := &Level{
		Width: width, Height: height, Depth: depth,
		Tiles: make([][]Tile, height),
		Monsters: make([]*Monster, 0), Items: make([]*Item, 0),
		Merchants: make([]*Merchant, 0), Altars: make([]*Altar, 0), Chests: make([]*Chest, 0),
		Rooms: make([]Room, 0), VisitCount: 1, logger: lgr,
		StairsUpX: -1, StairsUpY: -1, StairsDownX: -1, StairsDownY: -1,
		SecretStairsDownX: -1, SecretStairsDownY: -1, SecretStairsTargetDepth: 0,
		Traps: make([]*Trap, 0),
	}

	for y := 0; y < height; y++ {
		level.Tiles[y] = make([]Tile, width)
		for x := 0; x < width; x++ {
			level.Tiles[y][x] = Tile{Type: TileWall, Visible: false, Explored: false}
		}
	}

	level.generateDungeon()
	level.placeStairs(depth)
	
	// 🆕 ГЕНЕРАЦИЯ ЛОВУШЕК (количество равно глубине)
	trapCount := depth
	for i := 0; i < trapCount; i++ {
		x, y := level.FindFreeSpot()
		level.Traps = append(level.Traps, &Trap{X: x, Y: y, Triggered: false})
	}
	
	// 🆕 ГЕНЕРАЦИЯ СЕКРЕТНОЙ ЛЕСТНИЦЫ (появляется на 4 и 9 уровнях, ведет на Depth + 2)
	if depth == 4 || depth == 9 {
		x, y := level.FindFreeSpot()
		level.SecretStairsDownX = x
		level.SecretStairsDownY = y
		level.SecretStairsTargetDepth = depth + 2
	}

	level.spawnMonsters(5 + depth)
	level.spawnItems(8)
	
	// 🆕 НОВЫЕ ПРАВИЛА СПАВНА ТОРГОВЦЕВ И БОССОВ
	// Торговцы на уровнях: 2, 5, 8, 11, 14
	if depth == 2 || depth == 5 || depth == 8 || depth == 11 || depth == 14 {
		level.spawnMerchants(depth)
	}
	
	level.spawnAltars()
	level.spawnChests()
	
	// 🆕 БОССЫ КАЖДЫЕ 3 УРОВНЯ (3, 6, 9, 12, 15)
	if depth%3 == 0 && depth <= 15 {
		level.spawnBoss(depth)
	}

	if level.logger != nil {
		level.logger.Printf("LEVEL_NEW: depth=%d rooms=%d monsters=%d items=%d traps=%d secret=%v",
			depth, len(level.Rooms), len(level.Monsters), len(level.Items), len(level.Traps), level.SecretStairsTargetDepth > 0)
	}
	return level
}

func (l *Level) SetLogger(logger *log.Logger) {
	if l == nil { return }
	l.logger = logger
	for _, m := range l.Monsters {
		if m != nil { m.SetLogger(logger) }
	}
}
