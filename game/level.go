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

// Level — уровень подземелья.
type Level struct {
	Width    int        // ширина карты в клетках
	Height   int        // высота карты в клетках
	Depth    int        // глубина уровня (влияет на силу монстров)
	Tiles    [][]Tile   // двумерная карта клеток
	Monsters []*Monster // список живых монстров на уровне
	Items    []*Item    // список предметов на полу

	// ⚠️ НОВЫЕ ПОЛЯ: торговцы, алтари, сундуки
	Merchants []*Merchant // торговцы (на каждом 5-м уровне)
	Altars    []*Altar    // алтари (на каждом уровне)
	Chests    []*Chest    // сундуки (на каждом уровне)

	// 🆕 ЛОВУШКИ (ДОБАВИТЬ ЭТУ СТРОКУ)
	Traps []*Trap // список ловушек на уровне


	// 🆕 ЭТАП 2: Счётчик посещений уровня
	VisitCount int // количество посещений уровня (для удвоения цен)

	// Лестницы
	StairsUp    bool // есть ли лестница вверх
	StairsDown  bool // есть ли лестница вниз (всегда есть)
	StairsUpX   int  // координата X лестницы вверх
	StairsUpY   int  // координата Y лестницы вверх
	StairsDownX int  // координата X лестницы вниз
	StairsDownY int  // координата Y лестницы вниз

	// 🆕 СЕКРЕТНАЯ ЛЕСТНИЦА
	SecretStairsTargetDepth int // целевая глубина секретной лестницы (0 если нет)
	SecretStairsDownX       int // координата X секретной лестницы
	SecretStairsDownY       int // координата Y секретной лестницы

	// ⚠️ ВАЖНО: поле экспортируемое (с большой буквы),
	// чтобы json.Marshal сохранял его при сериализации.
	Rooms  []Room      `json:"Rooms"` // список сгенерированных комнат (для FindFreeSpot)
	logger *log.Logger `json:"-"`     // логгер для отладки (НЕ сохраняется в JSON)
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
