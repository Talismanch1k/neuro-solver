package resolution

// (Resolution Engine) — алгоритм автоматического доказательства теорем.

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const max_iterations = 500000

// ==========================================
// 1. Базовые структуры (Термы, Литералы)
// ==========================================

// Term — интерфейс для термов: переменных и констант
type Term interface {
	Name() string
	IsVariable() bool
	String() string
}

// Variable — переменная (одна строчная буква)
type Variable struct {
	name string
}

func NewVariable(name string) *Variable {
	return &Variable{name: name}
}

func (v *Variable) Name() string     { return v.name }
func (v *Variable) IsVariable() bool { return true }
func (v *Variable) String() string   { return v.name }

// Constant — константа (всё остальное)
type Constant struct {
	name string
}

func NewConstant(name string) *Constant {
	return &Constant{name: name}
}

func (c *Constant) Name() string     { return c.name }
func (c *Constant) IsVariable() bool { return false }
func (c *Constant) String() string   { return c.name }

// Literal — литерал (выражение над термами, может быть отрицательным)
type Literal struct {
	Predicate string
	Args      []Term
	Negated   bool
}

func NewLiteral(predicate string, args []Term, negated bool) *Literal {
	return &Literal{
		Predicate: predicate,
		Args:      args,
		Negated:   negated,
	}
}

func (l *Literal) String() string {
	prefix := ""
	if l.Negated {
		prefix = "¬"
	}
	argsStrs := make([]string, len(l.Args))
	for i, arg := range l.Args {
		argsStrs[i] = arg.String()
	}
	return fmt.Sprintf("%s%s(%s)", prefix, l.Predicate, strings.Join(argsStrs, ", "))
}

// Negate возвращает копию литерала с инвертированным знаком
func (l *Literal) Negate() *Literal {
	return NewLiteral(l.Predicate, l.Args, !l.Negated)
}

// Equal проверяет равенство двух литералов
func (l *Literal) Equal(other *Literal) bool {
	if l.Predicate != other.Predicate || l.Negated != other.Negated {
		return false
	}
	if len(l.Args) != len(other.Args) {
		return false
	}
	for i := range l.Args {
		if l.Args[i].Name() != other.Args[i].Name() {
			return false
		}
	}
	return true
}

// ==========================================
// 2. Клауза (с поддержкой ID)
// ==========================================

// Clause — клауза (дизъюнкция литералов)
type Clause struct {
	ID       int        // уникальный номер в списке всех клауз
	Literals []*Literal // список Литералов, которые содержит клауза
	Origin   string     // "init" (изначальна дана программе) или "res" (полученна лог. выводом)
	Parents  [2]*Clause // Родительские клаузы (для резольвент)
	Rule     string     // Описание унификации
}

func NewClause(id int, literals []*Literal, origin string, parents [2]*Clause, rule string) *Clause {
	// Удаление дубликатов и сортировка
	uniqueLiterals := removeDuplicateLiterals(literals)
	sort.Slice(uniqueLiterals, func(i, j int) bool {
		return uniqueLiterals[i].String() < uniqueLiterals[j].String()
	})

	return &Clause{
		ID:       id,
		Literals: uniqueLiterals,
		Origin:   origin,
		Parents:  parents,
		Rule:     rule,
	}
}

func (c *Clause) String() string {
	if len(c.Literals) == 0 {
		return "□" // Пустая клауза (Противоречие)
	}
	parts := make([]string, len(c.Literals))
	for i, lit := range c.Literals {
		parts[i] = lit.String()
	}
	return strings.Join(parts, " ∨ ")
}

// IsEmpty проверяет, является ли клауза пустой (противоречие)
func (c *Clause) IsEmpty() bool {
	return len(c.Literals) == 0
}

// Equal проверяет равенство двух клауз (по содержанию литералов)
func (c *Clause) Equal(other *Clause) bool {
	if len(c.Literals) != len(other.Literals) {
		return false
	}
	for i := range c.Literals {
		if !c.Literals[i].Equal(other.Literals[i]) {
			return false
		}
	}
	return true
}

// removeDuplicateLiterals удаляет дубликаты литералов к клаузе
func removeDuplicateLiterals(literals []*Literal) []*Literal {
	seen := make(map[string]bool)
	result := make([]*Literal, 0, len(literals))
	for _, lit := range literals {
		key := lit.String()
		if !seen[key] {
			seen[key] = true
			result = append(result, lit)
		}
	}
	return result
}

// ==========================================
// 3. Унификация
// ==========================================

// Theta — подстановка (отображение имени переменной на терм: x -> Const)
type Theta map[string]Term

// copyTheta создаёт копию подстановки
func copyTheta(original Theta) Theta {
	if original == nil {
		return make(Theta)
	}
	copied := make(Theta, len(original))
	for k, v := range original {
		copied[k] = v
	}
	return copied
}

// unify пытается унифицировать два объекта
// Возвращает nil, false если унификация невозможна
func unify(x, y interface{}, theta Theta) (Theta, bool) {
	if theta == nil {
		theta = make(Theta)
	}

	// Сравнение термов (переменных, констант)
	xTerm, xIsTerm := x.(Term)
	yTerm, yIsTerm := y.(Term)

	if xIsTerm && yIsTerm {
		// Оба — термы, если имена равны унификация не требуется
		if xTerm.Name() == yTerm.Name() {
			return theta, true
		}
		// если среди термов есть переменная -> унифицируем переменную
		if xTerm.IsVariable() {
			return unifyVar(xTerm, yTerm, theta)
		}
		if yTerm.IsVariable() {
			return unifyVar(yTerm, xTerm, theta)
		}
		// Оба константы, но разные — унификация невозможна
		return nil, false
	}

	// Сравнение литералов
	xLit, xIsLit := x.(*Literal)
	yLit, yIsLit := y.(*Literal)

	if xIsLit && yIsLit {
		// Если имена предикатов или длина списка арг. не совпали - унификация невозможна
		if xLit.Predicate != yLit.Predicate || len(xLit.Args) != len(yLit.Args) {
			return nil, false
		}
		// переходим к унификации списка аргументов
		return unifyLists(xLit.Args, yLit.Args, theta)
	}

	// Сравнение списков термов
	xList, xIsList := x.([]Term)
	yList, yIsList := y.([]Term)

	if xIsList && yIsList {
		// переводим к унификации списка аргуументов
		return unifyLists(xList, yList, theta)
	}

	return nil, false
}

// unifyLists унифицирует два списка термов
func unifyLists(xs, ys []Term, theta Theta) (Theta, bool) {
	// списки пусты - унификация завершена
	if len(xs) == 0 && len(ys) == 0 {
		return theta, true
	}
	// пуст только один список - унификация невозможна
	if len(xs) == 0 || len(ys) == 0 {
		return nil, false
	}

	// унифицируем первые элементы списка
	newTheta, ok := unify(xs[0], ys[0], theta)
	if !ok {
		return nil, false
	}
	// рекурсивный вызов, чтобы унифицировать хвост
	return unifyLists(xs[1:], ys[1:], newTheta)
}

// unifyVar унифицирует переменную с термом
func unifyVar(varTerm Term, x Term, theta Theta) (Theta, bool) {
	varName := varTerm.Name()

	// Если varTerm переменная уже в подстановке (например varTerm -> Const вызываем unify(theta[varTerm], const))
	if val, exists := theta[varName]; exists {
		return unify(val, x, theta)
	}

	// Если x — переменная и она в подстановке (например x -> Const вызываем unify(varTerm, theta[x]))
	if x.IsVariable() {
		if val, exists := theta[x.Name()]; exists {
			return unify(varTerm, val, theta)
		}
	}

	// Создаём новую подстановку (на этом этапе других подстанавок нет)
	newTheta := copyTheta(theta)
	newTheta[varName] = x
	return newTheta, true
}

// ==========================================
// 4. Движок Резолюций (Resolution Engine)
// ==========================================

// ResolutionEngine — движок для поиска доказательств методом резолюций
type ResolutionEngine struct {
	clauses       []*Clause
	clauseCounter int
}

// NewResolutionEngine создаёт новый движок резолюций
func NewResolutionEngine() *ResolutionEngine {
	return &ResolutionEngine{
		clauses:       make([]*Clause, 0),
		clauseCounter: 1,
	}
}

// getNextID возвращает следующий уникальный ID для клаузы
func (e *ResolutionEngine) getNextID() int {
	id := e.clauseCounter
	e.clauseCounter++
	return id
}

// ParseInput преобразует строки в объекты Clause (использует парсер)
func (e *ResolutionEngine) ParseInput(inputStrings []string) {
	e.clauses = make([]*Clause, 0)
	e.clauseCounter = 1

	for _, s := range inputStrings {
		literalsStr := strings.Split(s, "∨")
		literals := make([]*Literal, 0)

		for _, lStr := range literalsStr {
			lStr = strings.TrimSpace(lStr)
			negated := strings.HasPrefix(lStr, "¬")
			if negated {
				lStr = strings.TrimPrefix(lStr, "¬")
			}

			// Парсим литерал вручную
			name, args := parseLiteral(lStr)
			if name != "" {
				literals = append(literals, NewLiteral(name, args, negated))
			}
		}

		// Создаём начальную клаузу
		newClause := NewClause(e.getNextID(), literals, "init", [2]*Clause{}, "")
		e.clauses = append(e.clauses, newClause)
	}
}

// parseLiteral парсит литерал вида "Predicate(arg1, arg2, ...)"
func parseLiteral(s string) (string, []Term) {
	// Ищем открывающую скобку
	openIdx := strings.Index(s, "(")
	if openIdx == -1 {
		return "", nil
	}
	// Ищем закрывающую скобку
	closeIdx := strings.LastIndex(s, ")")
	if closeIdx == -1 || closeIdx <= openIdx {
		return "", nil
	}

	name := s[:openIdx]
	argsStr := s[openIdx+1 : closeIdx]
	argsRaw := strings.Split(argsStr, ",")
	args := make([]Term, 0, len(argsRaw))

	for _, a := range argsRaw {
		a = strings.TrimSpace(a)
		// Логика: 1 руна строчная буква = переменная, иначе константа
		if isSingleLowerLetter(a) {
			args = append(args, NewVariable(a))
		} else {
			args = append(args, NewConstant(a))
		}
	}

	return name, args
}

// isSingleLowerLetter проверяет, является ли строка одной строчной буквой
func isSingleLowerLetter(s string) bool {
	runes := []rune(s)
	if len(runes) != 1 {
		return false
	}
	return unicode.IsLower(runes[0]) && unicode.IsLetter(runes[0])
}

// substitute применяет подстановку к литералу
func (e *ResolutionEngine) substitute(lit *Literal, theta Theta) *Literal {
	newArgs := make([]Term, len(lit.Args))
	for i, arg := range lit.Args {
		val := arg
		// подстановка по всем переменных на основе значения в theta
		for val.IsVariable() {
			if newVal, exists := theta[val.Name()]; exists {
				val = newVal
			} else {
				break
			}
		}
		newArgs[i] = val
	}
	return NewLiteral(lit.Predicate, newArgs, lit.Negated)
}

// resolvePair возвращает ВСЕ возможные резольвенты из двух клауз
func (e *ResolutionEngine) resolvePair(c1, c2 *Clause) []*Clause {
	var resolvents []*Clause

	for i, l1 := range c1.Literals {
		for j, l2 := range c2.Literals {
			// Ищем пару L и ¬L
			if l1.Predicate == l2.Predicate && l1.Negated != l2.Negated {
				// Пытаемся унифицировать (l1 и инверсию l2)
				theta, ok := unify(l1, l2.Negate(), nil)

				if ok {
					// Собираем новые литералы
					newLits := make([]*Literal, 0)

					// Все из c1 кроме l1 (по индексу, чтобы избежать проблем с дубликатами)
					for idx, l := range c1.Literals {
						if idx != i {
							newLits = append(newLits, e.substitute(l, theta))
						}
					}
					// Все из c2 кроме l2
					for idx, l := range c2.Literals {
						if idx != j {
							newLits = append(newLits, e.substitute(l, theta))
						}
					}

					// Формируем строку унификации для лога
					unifStr := formatTheta(theta)
					if unifStr == "" {
						unifStr = "(пустая)"
					}

					resolvent := NewClause(
						e.getNextID(),
						newLits,
						"res",
						[2]*Clause{c1, c2},
						fmt.Sprintf("Унификация %s", unifStr),
					)
					resolvents = append(resolvents, resolvent)
				}
			}
		}
	}
	return resolvents
}

// formatTheta форматирует подстановку для вывода
func formatTheta(theta Theta) string {
	if len(theta) == 0 {
		return ""
	}
	parts := make([]string, 0, len(theta))
	for k, v := range theta {
		parts = append(parts, fmt.Sprintf("(%s|%s)", v.String(), k))
	}
	sort.Strings(parts) // Для детерминированного вывода
	return strings.Join(parts, ", ")
}

// ProofResult — результат доказательства с двумя видами логов
type ProofResult struct {
	Success  bool
	FullLog  string // Полный лог со всеми резолюциями
	ShortLog string // Краткий лог — только цепочка к противоречию
}

// buildProofChain восстанавливает цепочку клауз, приведших к противоречию
func (e *ResolutionEngine) buildProofChain(contradiction *Clause) []*Clause {
	chain := make([]*Clause, 0)
	visited := make(map[int]bool)

	var collect func(c *Clause)
	collect = func(c *Clause) {
		if c == nil || visited[c.ID] {
			return
		}
		visited[c.ID] = true

		// Сначала собираем родителей (рекурсивно)
		if c.Origin == "res" {
			collect(c.Parents[0])
			collect(c.Parents[1])
		}
		// Затем добавляем текущую клаузу
		chain = append(chain, c)
	}

	collect(contradiction)
	return chain
}

// formatShortLog форматирует краткий лог по цепочке доказательства
func (e *ResolutionEngine) formatShortLog(chain []*Clause) string {
	var lines []string
	lines = append(lines, "=== КРАТКИЙ ЛОГ (цепочка доказательства) ===\n")

	// Сначала выводим начальные клаузы из цепочки
	lines = append(lines, "Используемые начальные клаузы:")
	for _, c := range chain {
		if c.Origin == "init" {
			lines = append(lines, fmt.Sprintf("  [%d] %s", c.ID, c.String()))
		}
	}

	// Затем выводим шаги резолюции
	lines = append(lines, "\nШаги резолюции:")
	stepNum := 1
	for _, c := range chain {
		if c.Origin == "res" {
			// Определяем тип шага
			stepType := "Резолюция"
			if c.IsEmpty() {
				stepType = "Противоречие найдено"
			}

			stepLog := fmt.Sprintf(
				"\nШаг %d - %s\n    Клауза 1: [%d] %s\n    Клауза 2: [%d] %s\n    Действие: %s\n    Результат: [%d] %s",
				stepNum,
				stepType,
				c.Parents[0].ID, c.Parents[0].String(),
				c.Parents[1].ID, c.Parents[1].String(),
				c.Rule,
				c.ID, c.String(),
			)
			lines = append(lines, stepLog)
			stepNum++
		}
	}

	lines = append(lines, "\nРезультат: резолюция успешна. (обнаружено противоречие □)")
	return strings.Join(lines, "\n")
}

// Prove запускает поиск доказательства с генерацией логов
func (e *ResolutionEngine) Prove() ProofResult {
	activeClauses := make([]*Clause, len(e.clauses))
	copy(activeClauses, e.clauses)

	processedPairs := make(map[[2]int]bool)

	// Формирование начальной части полного лога
	var logLines []string
	logLines = append(logLines, "=== ПОЛНЫЙ ЛОГ (все резолюции) ===\n")
	logLines = append(logLines, fmt.Sprintf("Начальные клаузы: %d", len(activeClauses)))
	for _, c := range activeClauses {
		logLines = append(logLines, fmt.Sprintf("  [%d] %s", c.ID, c.String()))
	}

	stepCount := 1
	processedChecks := 0

	// сам алгоритм резолюций
	for {
		progress := false
		// Копируем список, так как он будет расти
		currentPool := make([]*Clause, len(activeClauses))
		copy(currentPool, activeClauses)

		for i := 0; i < len(currentPool); i++ {
			for j := i + 1; j < len(currentPool); j++ {
				processedChecks++

				// превышено допустимое число итераций
				if processedChecks > max_iterations {
					return ProofResult{Success: false, FullLog: strings.Join(logLines, ""), ShortLog: fmt.Sprintf("Не удалось найти решение за отведенное число итераций: %d.", max_iterations)}
				}

				c1 := currentPool[i]
				c2 := currentPool[j]

				// Избегаем повторной обработки пары
				pairID := [2]int{c1.ID, c2.ID}
				if c1.ID > c2.ID {
					pairID = [2]int{c2.ID, c1.ID}
				}
				// если пара уже встречалась, пропускаем её
				if processedPairs[pairID] {
					continue
				}
				processedPairs[pairID] = true

				// для данной пары получаем ВСЕ резольвенты
				resolvents := e.resolvePair(c1, c2)

				for _, resolvent := range resolvents {
					// Проверка на дубликаты
					isDuplicate := false
					for _, existing := range activeClauses {
						if resolvent.Equal(existing) {
							isDuplicate = true
							break
						}
					}

					// если полученная резольвента новая, добавляем её в список активных клауз
					if !isDuplicate {
						activeClauses = append(activeClauses, resolvent)
						progress = true

						// Формирование лога шага
						isContradiction := resolvent.IsEmpty()
						stepName := fmt.Sprintf("Шаг %d - ", stepCount)
						if isContradiction {
							stepName += "Противоречие найдено"
						} else {
							stepName += "Резолюция"
						}

						stepLog := fmt.Sprintf(
							"\n%s\n    Клауза 1: [%d] %s\n    Клауза 2: [%d] %s\n    Действие: %s\n    Результат: [%d] %s",
							stepName,
							c1.ID, c1.String(),
							c2.ID, c2.String(),
							resolvent.Rule,
							resolvent.ID, resolvent.String(),
						)
						logLines = append(logLines, stepLog)
						stepCount++

						// если нашли противоречие - завершаем программу
						if isContradiction {
							logLines = append(logLines, "\nРезультат: Получено противоречие → резолюция успешна.")

							// Строим краткий лог
							chain := e.buildProofChain(resolvent)
							shortLog := e.formatShortLog(chain)

							return ProofResult{
								Success:  true,
								FullLog:  strings.Join(logLines, "\n"),
								ShortLog: shortLog,
							}
						}
					}
				}
			}
		}

		// среди множества пар не удалось получить резольвенту -> вывода не существет
		if !progress {
			logLines = append(logLines, "\nРезультат: Не удалось получить противоречие (новые клаузы больше не выводятся).")
			return ProofResult{
				Success:  false,
				FullLog:  strings.Join(logLines, "\n"),
				ShortLog: "Доказательство не найдено — краткий лог недоступен.",
			}
		}
	}
}
