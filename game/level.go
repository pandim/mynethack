package game

import (
	"log"
)

// =============================================================================
// КОНСТАНТЫ ТИПОВ КЛЕТОК
// =============================================================================
const (
	TileWall  = 0 // Стена — непроходимая клетка
	TileFloor = 1 // Пол — проходимая клетка
)

// =============================================================================
// КОНСТАНТЫ УРОВНЯ
// =============================================================================
const (
	// PlayerFOVRadius — радиус поля зрения игрока в клетках.
	PlayerFOVRadius = 8
)

// =============================================================================
// СТРУКТУРЫ ДАННЫХ
// =============================================================================
// Tile — одна клетка карты.
// Хранит тип клетки и информацию о её видимости/исследованности.
type Tile struct {
	Type     int  // TileWall или TileFloor
	Visible  bool // Видна ли клетка прямо сейчас (в радиусе зрения игрока)
	Explored bool // Была ли клетка когда-либо видна (для отображения тумана войны)
}

// Room — прямоугольная комната в подземелье.
// Используется при генерации уровня.
type Room struct {
	X, Y, W, H int // координаты левого верхнего угла, ширина и высота
}

// point — точка на карте (вспомогательный тип).
type point struct {
	X, Y int
}

// Level — уровень подземелья.
// Содержит карту, монстров, предметы, лестницы и логгер.
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

	// Лестницы
	StairsUp    bool // есть ли лестница вверх
	StairsDown  bool // есть ли лестница вниз (всегда есть)
	StairsUpX   int  // координата X лестницы вверх
	StairsUpY   int  // координата Y лестницы вверх
	StairsDownX int  // координата X лестницы вниз
	StairsDownY int  // координата Y лестницы вниз

	// ⚠️ ВАЖНО: поле экспортируемое (с большой буквы),
	// чтобы json.Marshal сохранял его при сериализации.
	Rooms  []Room      `json:"Rooms"` // список сгенерированных комнат (для FindFreeSpot)
	logger *log.Logger `json:"-"`     // логгер для отладки (НЕ сохраняется в JSON)
}

// =============================================================================
// СОЗДАНИЕ НОВОГО УРОВНЯ
// =============================================================================
// NewLevel создаёт новый уровень подземелья заданного размера и глубины.
//
// Принимает опциональный логгер:
//
//	NewLevel(width, height, depth)
//	NewLevel(width, height, depth, logger)
//
// Порядок генерации:
//   1. Заполняем всю карту стенами
//   2. Генерируем комнаты и коридоры (generateDungeon)
//   3. Размещаем лестницы (placeStairs)
//   4. Спавним монстров (spawnMonsters) — количество зависит от глубины
//   5. Спавним предметы (spawnItems)
//   6. Спавним торговцев (на каждом 5-м уровне)
//   7. Спавним алтари (на каждом уровне)
//   8. Спавним сундуки (на каждом уровне)
func NewLevel(width, height int, depth int, logger ...*log.Logger) *Level {
	var lgr *log.Logger
	if len(logger) > 0 {
		lgr = logger[0]
	}

	// Защита от некорректных размеров
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if depth < 1 {
		depth = 1
	}

	// Создаём структуру уровня
	level := &Level{
		Width:       width,
		Height:      height,
		Depth:       depth,
		Tiles:       make([][]Tile, height),
		Monsters:    make([]*Monster, 0),
		Items:       make([]*Item, 0),
		Merchants:   make([]*Merchant, 0),
		Altars:      make([]*Altar, 0),
		Chests:      make([]*Chest, 0),
		Rooms:       make([]Room, 0),
		logger:      lgr,
		StairsUpX:   -1, // -1 означает "не размещена"
		StairsUpY:   -1,
		StairsDownX: -1,
		StairsDownY: -1,
	}

	// Инициализируем карту: все клетки — стены, ничего не видно
	for y := 0; y < height; y++ {
		level.Tiles[y] = make([]Tile, width)
		for x := 0; x < width; x++ {
			level.Tiles[y][x] = Tile{
				Type:     TileWall,
				Visible:  false,
				Explored: false,
			}
		}
	}

	// Генерируем подземелье: комнаты, коридоры, лестницы, монстры, предметы
	// Функции определены в level_gen.go, level_stairs.go, level_spawn.go
	level.generateDungeon()
	level.placeStairs(depth)
	level.spawnMonsters(5 + depth) // чем глубже — тем больше монстров
	level.spawnItems(8)            // фиксированное количество предметов
	level.spawnMerchants(depth)    // торговцы на каждом 5-м уровне
	level.spawnAltars()            // алтари на каждом уровне
	level.spawnChests()            // сундуки на каждом уровне
	level.spawnBoss(depth)         // боссы на каждом 6-м уровне
	// 🆕 Новые спавны Этапа 1
	level.spawnRelics(depth)         // реликвии для продажи
	level.spawnScrolls(depth)        // свитки с заклинаниями
	level.spawnKeys(depth)           // ключи от золотых сундуков
	level.spawnGoldenChests(depth)   // золотые сундуки (уровни 3+)

	// Логируем параметры сгенерированного уровня
	if level.logger != nil {
		level.logger.Printf(
			"LEVEL_NEW: depth=%d upstairs=(%d,%d) downstairs=(%d,%d) rooms=%d monsters=%d items=%d merchants=%d altars=%d chests=%d",
			depth,
			level.StairsUpX, level.StairsUpY,
			level.StairsDownX, level.StairsDownY,
			len(level.Rooms), len(level.Monsters), len(level.Items),
			len(level.Merchants), len(level.Altars), len(level.Chests),
		)
	}

	return level
}

// SetLogger — устанавливает логгер для уровня и всех его монстров.
// Вызывается после загрузки уровня из сохранения.
func (l *Level) SetLogger(logger *log.Logger) {
	if l == nil {
		return
	}
	l.logger = logger
	for _, m := range l.Monsters {
		if m != nil {
			m.SetLogger(logger)
		}
	}
}