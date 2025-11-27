package resolution

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const max_iterations = 500000

// ==========================================
// 1. Базовые структуры (Термы)
// ==========================================

// Term — интерфейс. Обязателен метод ContainsVar для проверки зацикливания.
type Term interface {
	Name() string
	IsVariable() bool
	String() string
	ContainsVar(varName string) bool
}

// Variable — переменная (x, y, z...)
type Variable struct {
	name string
}

func NewVariable(name string) *Variable { return &Variable{name: name} }
func (v *Variable) Name() string        { return v.name }
func (v *Variable) IsVariable() bool    { return true }
func (v *Variable) String() string      { return v.name }
func (v *Variable) ContainsVar(name string) bool {
	return v.name == name
}

// Constant — константа (a, Bob, 1).
type Constant struct {
	name string
}

func NewConstant(name string) *Constant { return &Constant{name: name} }
func (c *Constant) Name() string        { return c.name }
func (c *Constant) IsVariable() bool    { return false }
func (c *Constant) String() string      { return c.name }
func (c *Constant) ContainsVar(name string) bool {
	return false
}

// Function — функциональный терм: f(x), Отец(x)
type Function struct {
	name string
	args []Term
}

func NewFunction(name string, args []Term) *Function {
	return &Function{name: name, args: args}
}
func (f *Function) Name() string     { return f.name }
func (f *Function) IsVariable() bool { return false }
func (f *Function) String() string {
	parts := make([]string, len(f.args))
	for i, arg := range f.args {
		parts[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.name, strings.Join(parts, ", "))
}

// Проверка вхождения: рекурсивно ищем переменную в аргументах функции
func (f *Function) ContainsVar(name string) bool {
	for _, arg := range f.args {
		if arg.ContainsVar(name) {
			return true
		}
	}
	return false
}

// ==========================================
// 2. Литерал и Клауза
// ==========================================

type Literal struct {
	Predicate string
	Args      []Term
	Negated   bool
}

func NewLiteral(predicate string, args []Term, negated bool) *Literal {
	return &Literal{Predicate: predicate, Args: args, Negated: negated}
}

func (l *Literal) String() string {
	prefix := ""
	if l.Negated {
		prefix = "¬"
	}
	parts := make([]string, len(l.Args))
	for i, arg := range l.Args {
		parts[i] = arg.String()
	}
	return fmt.Sprintf("%s%s(%s)", prefix, l.Predicate, strings.Join(parts, ", "))
}

func (l *Literal) Negate() *Literal {
	return NewLiteral(l.Predicate, l.Args, !l.Negated)
}

// Equal использует строковое представление для глубокого сравнения
func (l *Literal) Equal(other *Literal) bool {
	if l.Predicate != other.Predicate || l.Negated != other.Negated {
		return false
	}
	if len(l.Args) != len(other.Args) {
		return false
	}
	for i := range l.Args {
		if l.Args[i].String() != other.Args[i].String() {
			return false
		}
	}
	return true
}

type Clause struct {
	ID       int
	Literals []*Literal
	Origin   string
	Parents  [2]*Clause
	Rule     string
}

func NewClause(id int, literals []*Literal, origin string, parents [2]*Clause, rule string) *Clause {
	uniqueLiterals := removeDuplicateLiterals(literals)
	// Сортировка для детерминизма
	sort.Slice(uniqueLiterals, func(i, j int) bool {
		return uniqueLiterals[i].String() < uniqueLiterals[j].String()
	})
	return &Clause{ID: id, Literals: uniqueLiterals, Origin: origin, Parents: parents, Rule: rule}
}

func (c *Clause) String() string {
	if len(c.Literals) == 0 {
		return "□"
	}
	parts := make([]string, len(c.Literals))
	for i, lit := range c.Literals {
		parts[i] = lit.String()
	}
	return strings.Join(parts, " ∨ ")
}

func (c *Clause) IsEmpty() bool { return len(c.Literals) == 0 }

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
// 3. Унификация (Ядро логики)
// ==========================================

type Theta map[string]Term

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

func unify(x, y interface{}, theta Theta) (Theta, bool) {
	if theta == nil {
		theta = make(Theta)
	}

	xTerm, xIsTerm := x.(Term)
	yTerm, yIsTerm := y.(Term)

	if xIsTerm && yIsTerm {
		// Оптимизация: если строки равны, термы идентичны
		if xTerm.String() == yTerm.String() {
			return theta, true
		}

		// Если один из них переменная
		if xTerm.IsVariable() {
			return unifyVar(xTerm, yTerm, theta)
		}
		if yTerm.IsVariable() {
			return unifyVar(yTerm, xTerm, theta)
		}

		// Если оба — Функции
		xFunc, xIsFunc := xTerm.(*Function)
		yFunc, yIsFunc := yTerm.(*Function)
		if xIsFunc && yIsFunc {
			if xFunc.name != yFunc.name || len(xFunc.args) != len(yFunc.args) {
				return nil, false
			}
			return unifyLists(xFunc.args, yFunc.args, theta)
		}

		// Разные типы (Константа vs Функция) или разные константы
		return nil, false
	}

	// Литералы
	xLit, xIsLit := x.(*Literal)
	yLit, yIsLit := y.(*Literal)
	if xIsLit && yIsLit {
		if xLit.Predicate != yLit.Predicate || len(xLit.Args) != len(yLit.Args) {
			return nil, false
		}
		return unifyLists(xLit.Args, yLit.Args, theta)
	}

	return nil, false
}

func unifyLists(xs, ys []Term, theta Theta) (Theta, bool) {
	if len(xs) == 0 && len(ys) == 0 {
		return theta, true
	}
	if len(xs) == 0 || len(ys) == 0 {
		return nil, false
	}
	newTheta, ok := unify(xs[0], ys[0], theta)
	if !ok {
		return nil, false
	}
	return unifyLists(xs[1:], ys[1:], newTheta)
}

func unifyVar(varTerm Term, x Term, theta Theta) (Theta, bool) {
	varName := varTerm.Name()

	// 1. Проверяем, связана ли уже переменная varTerm
	if val, exists := theta[varName]; exists {
		return unify(val, x, theta)
	}

	// 2. Проверяем, связана ли x (если x тоже переменная)
	if x.IsVariable() {
		if val, exists := theta[x.Name()]; exists {
			return unify(varTerm, val, theta)
		}
	}

	// 3. Occurs Check: Нельзя связать x = f(x)
	if x.ContainsVar(varName) {
		return nil, false
	}

	// 4. Связываем
	newTheta := copyTheta(theta)
	newTheta[varName] = x
	return newTheta, true
}

// ==========================================
// 4. Парсер (Умный парсер скобок)
// ==========================================

type ResolutionEngine struct {
	clauses       []*Clause
	clauseCounter int
}

func NewResolutionEngine() *ResolutionEngine {
	return &ResolutionEngine{
		clauses:       make([]*Clause, 0),
		clauseCounter: 1,
	}
}

func (e *ResolutionEngine) getNextID() int {
	id := e.clauseCounter
	e.clauseCounter++
	return id
}

func (e *ResolutionEngine) ParseInput(inputStrings []string) {
	e.clauses = make([]*Clause, 0)
	e.clauseCounter = 1

	for _, s := range inputStrings {
		// Разделяем по ИЛИ
		literalsStr := strings.Split(s, "∨")
		literals := make([]*Literal, 0)

		for _, lStr := range literalsStr {
			lStr = strings.TrimSpace(lStr)
			negated := false

			// Корректная обработка знака отрицания
			if strings.HasPrefix(lStr, "¬") {
				negated = true
				runes := []rune(lStr)
				lStr = string(runes[1:])
			}

			name, args := parseLiteralString(lStr)
			if name != "" {
				literals = append(literals, NewLiteral(name, args, negated))
			}
		}

		newClause := NewClause(e.getNextID(), literals, "init", [2]*Clause{}, "")
		e.clauses = append(e.clauses, newClause)
	}
}

// parseLiteralString выделяет P и (args...)
func parseLiteralString(s string) (string, []Term) {
	s = strings.TrimSpace(s)
	openIdx := strings.Index(s, "(")
	if openIdx == -1 {
		return "", nil
	}
	closeIdx := strings.LastIndex(s, ")")
	if closeIdx == -1 || closeIdx <= openIdx {
		return "", nil
	}

	name := s[:openIdx]
	argsBody := s[openIdx+1 : closeIdx]
	args := parseArgs(argsBody)
	return name, args
}

// parseArgs рекурсивно парсит список аргументов, учитывая запятые внутри функций
func parseArgs(s string) []Term {
	var args []Term
	var currentToken strings.Builder
	depth := 0

	runes := []rune(s)
	for i, r := range runes {
		switch r {
		case '(':
			depth++
			currentToken.WriteRune(r)
		case ')':
			depth--
			currentToken.WriteRune(r)
		case ',':
			if depth == 0 {
				termStr := strings.TrimSpace(currentToken.String())
				if termStr != "" {
					args = append(args, parseTerm(termStr))
				}
				currentToken.Reset()
			} else {
				currentToken.WriteRune(r)
			}
		default:
			currentToken.WriteRune(r)
		}

		if i == len(runes)-1 {
			termStr := strings.TrimSpace(currentToken.String())
			if termStr != "" {
				args = append(args, parseTerm(termStr))
			}
		}
	}
	return args
}

// parseTerm определяет тип терма: Переменная, Функция или Константа
func parseTerm(s string) Term {
	s = strings.TrimSpace(s)
	// Это Функция?
	openIdx := strings.Index(s, "(")
	if openIdx != -1 && strings.HasSuffix(s, ")") {
		funcName := s[:openIdx]
		argsBody := s[openIdx+1 : len(s)-1]
		args := parseArgs(argsBody) // Рекурсия
		return NewFunction(funcName, args)
	}

	// Переменная или Константа?
	if isSingleLowerLetter(s) {
		return NewVariable(s)
	}
	return NewConstant(s)
}

// isSingleLowerLetter: переменные - только одна строчная буква (по ТЗ промпта)
func isSingleLowerLetter(s string) bool {
	runes := []rune(s)
	if len(runes) != 1 {
		return false
	}
	return unicode.IsLower(runes[0]) && unicode.IsLetter(runes[0])
}

// ==========================================
// 5. Подстановка и Резолюция
// ==========================================

func (e *ResolutionEngine) substitute(lit *Literal, theta Theta) *Literal {
	newArgs := make([]Term, len(lit.Args))
	for i, arg := range lit.Args {
		newArgs[i] = e.applyThetaToTermSafe(arg, theta, make(map[string]bool))
	}
	return NewLiteral(lit.Predicate, newArgs, lit.Negated)
}

// applyThetaToTermSafe применяет подстановку с защитой от бесконечной рекурсии
func (e *ResolutionEngine) applyThetaToTermSafe(t Term, theta Theta, visited map[string]bool) Term {
	// 1. Переменная: ищем замену
	if t.IsVariable() {
		varName := t.Name()
		// Защита от циклической подстановки
		if visited[varName] {
			return t
		}
		if val, exists := theta[varName]; exists {
			// Отмечаем переменную как посещённую
			visited[varName] = true
			result := e.applyThetaToTermSafe(val, theta, visited)
			delete(visited, varName) // Убираем после обработки для других путей
			return result
		}
		return t
	}
	// 2. Функция: заходим внутрь
	if f, ok := t.(*Function); ok {
		newFnArgs := make([]Term, len(f.args))
		for i, arg := range f.args {
			newFnArgs[i] = e.applyThetaToTermSafe(arg, theta, visited)
		}
		return NewFunction(f.name, newFnArgs)
	}
	// 3. Константа: без изменений
	return t
}

// applyThetaToTerm - обёртка для совместимости
func (e *ResolutionEngine) applyThetaToTerm(t Term, theta Theta) Term {
	return e.applyThetaToTermSafe(t, theta, make(map[string]bool))
}

func (e *ResolutionEngine) resolvePair(c1, c2 *Clause) []*Clause {
	var resolvents []*Clause

	for i, l1 := range c1.Literals {
		for j, l2 := range c2.Literals {
			// Ищем контрарную пару
			if l1.Predicate == l2.Predicate && l1.Negated != l2.Negated {
				// Пытаемся унифицировать
				theta, ok := unify(l1, l2.Negate(), nil)

				if ok {
					newLits := make([]*Literal, 0)
					// Копируем и подставляем остальные литералы
					for idx, l := range c1.Literals {
						if idx != i {
							newLits = append(newLits, e.substitute(l, theta))
						}
					}
					for idx, l := range c2.Literals {
						if idx != j {
							newLits = append(newLits, e.substitute(l, theta))
						}
					}

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

func formatTheta(theta Theta) string {
	if len(theta) == 0 {
		return ""
	}
	parts := make([]string, 0, len(theta))
	for k, v := range theta {
		parts = append(parts, fmt.Sprintf("%s/%s", v.String(), k))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// ==========================================
// 6. Вывод и логирование (Без изменений)
// ==========================================

type ProofResult struct {
	Success  bool
	FullLog  string
	ShortLog string
}

func (e *ResolutionEngine) buildProofChain(contradiction *Clause) []*Clause {
	chain := make([]*Clause, 0)
	visited := make(map[int]bool)

	var collect func(c *Clause)
	collect = func(c *Clause) {
		if c == nil || visited[c.ID] {
			return
		}
		visited[c.ID] = true
		if c.Origin == "res" {
			collect(c.Parents[0])
			collect(c.Parents[1])
		}
		chain = append(chain, c)
	}
	collect(contradiction)
	return chain
}

func (e *ResolutionEngine) formatShortLog(chain []*Clause) string {
	var lines []string
	lines = append(lines, "=== КРАТКИЙ ЛОГ (цепочка доказательства) ===\n")
	lines = append(lines, "Используемые начальные клаузы:")
	for _, c := range chain {
		if c.Origin == "init" {
			lines = append(lines, fmt.Sprintf("  [%d] %s", c.ID, c.String()))
		}
	}
	lines = append(lines, "\nШаги резолюции:")
	stepNum := 1
	for _, c := range chain {
		if c.Origin == "res" {
			stepType := "Резолюция"
			if c.IsEmpty() {
				stepType = "Противоречие найдено"
			}
			stepLog := fmt.Sprintf(
				"\nШаг %d - %s\n    Клауза 1: [%d] %s\n    Клауза 2: [%d] %s\n    Действие: %s\n    Результат: [%d] %s",
				stepNum, stepType,
				c.Parents[0].ID, c.Parents[0].String(),
				c.Parents[1].ID, c.Parents[1].String(),
				c.Rule, c.ID, c.String(),
			)
			lines = append(lines, stepLog)
			stepNum++
		}
	}
	lines = append(lines, "\nРезультат: резолюция успешна (□).")
	return strings.Join(lines, "\n")
}

func (e *ResolutionEngine) Prove() ProofResult {
	activeClauses := make([]*Clause, len(e.clauses))
	copy(activeClauses, e.clauses)
	processedPairs := make(map[[2]int]bool)

	var logLines []string
	logLines = append(logLines, "=== ПОЛНЫЙ ЛОГ (все резолюции) ===\n")
	logLines = append(logLines, fmt.Sprintf("Начальные клаузы: %d", len(activeClauses)))
	for _, c := range activeClauses {
		logLines = append(logLines, fmt.Sprintf("  [%d] %s", c.ID, c.String()))
	}

	stepCount := 1
	processedChecks := 0

	for {
		progress := false
		currentPool := make([]*Clause, len(activeClauses))
		copy(currentPool, activeClauses)

		for i := 0; i < len(currentPool); i++ {
			for j := i + 1; j < len(currentPool); j++ {
				processedChecks++
				if processedChecks > max_iterations {
					return ProofResult{Success: false, FullLog: strings.Join(logLines, "\n"), ShortLog: "TIMEOUT"}
				}

				c1 := currentPool[i]
				c2 := currentPool[j]
				pairID := [2]int{c1.ID, c2.ID}
				if c1.ID > c2.ID {
					pairID = [2]int{c2.ID, c1.ID}
				}

				if processedPairs[pairID] {
					continue
				}
				processedPairs[pairID] = true

				resolvents := e.resolvePair(c1, c2)

				for _, resolvent := range resolvents {
					isDuplicate := false
					for _, existing := range activeClauses {
						if resolvent.Equal(existing) {
							isDuplicate = true
							break
						}
					}

					if !isDuplicate {
						activeClauses = append(activeClauses, resolvent)
						progress = true

						isContradiction := resolvent.IsEmpty()
						stepName := fmt.Sprintf("Шаг %d - ", stepCount)
						if isContradiction {
							stepName += "Противоречие найдено"
						} else {
							stepName += "Резолюция"
						}

						stepLog := fmt.Sprintf(
							"\n%s\n    Клауза 1: [%d] %s\n    Клауза 2: [%d] %s\n    Действие: %s\n    Результат: [%d] %s",
							stepName, c1.ID, c1.String(), c2.ID, c2.String(), resolvent.Rule, resolvent.ID, resolvent.String(),
						)
						logLines = append(logLines, stepLog)
						stepCount++

						if isContradiction {
							logLines = append(logLines, "\nРезультат: Доказано (□).")
							chain := e.buildProofChain(resolvent)
							shortLog := e.formatShortLog(chain)
							return ProofResult{Success: true, FullLog: strings.Join(logLines, "\n"), ShortLog: shortLog}
						}
					}
				}
			}
		}

		if !progress {
			logLines = append(logLines, "\nРезультат: Противоречие не найдено (база непротиворечива).")
			return ProofResult{Success: false, FullLog: strings.Join(logLines, "\n"), ShortLog: strings.Join(logLines, "\n")}
		}
	}
}
