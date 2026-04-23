package agent

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Node interface {
	Pos() Position
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Workflow *WorkflowDecl `json:"workflow,omitempty"`
}

func (p *Program) Pos() Position {
	if p == nil || p.Workflow == nil {
		return Position{}
	}
	return p.Workflow.Pos()
}

type WorkflowDecl struct {
	Name     string      `json:"name"`
	Trigger  *OnDecl     `json:"trigger,omitempty"`
	Body     []Statement `json:"body,omitempty"`
	Position Position    `json:"position"`
}

func (n *WorkflowDecl) Pos() Position { return n.Position }

type OnDecl struct {
	Event    string   `json:"event"`
	Position Position `json:"position"`
}

func (n *OnDecl) Pos() Position { return n.Position }

type IfStatement struct {
	Condition Expression  `json:"condition"`
	Body      []Statement `json:"body,omitempty"`
	Position  Position    `json:"position"`
}

func (n *IfStatement) Pos() Position  { return n.Position }
func (n *IfStatement) statementNode() {}

type SetStatement struct {
	Target   *IdentifierExpr `json:"target"`
	Value    Expression      `json:"value"`
	Position Position        `json:"position"`
}

func (n *SetStatement) Pos() Position  { return n.Position }
func (n *SetStatement) statementNode() {}

type NotifyStatement struct {
	Target   *IdentifierExpr `json:"target"`
	Value    Expression      `json:"value"`
	Position Position        `json:"position"`
}

func (n *NotifyStatement) Pos() Position  { return n.Position }
func (n *NotifyStatement) statementNode() {}

//nolint:revive // AgentStatement is intentionally named in the agent package for DSL clarity
type AgentStatement struct {
	Name     *IdentifierExpr `json:"name"`
	Input    Expression      `json:"input,omitempty"`
	Position Position        `json:"position"`
}

func (n *AgentStatement) Pos() Position  { return n.Position }
func (n *AgentStatement) statementNode() {}

type DispatchStatement struct {
	Target   *IdentifierExpr `json:"target"`
	Payload  Expression      `json:"payload,omitempty"`
	Position Position        `json:"position"`
}

func (n *DispatchStatement) Pos() Position  { return n.Position }
func (n *DispatchStatement) statementNode() {}

type SurfaceStatement struct {
	Entity   *IdentifierExpr `json:"entity"`
	View     *IdentifierExpr `json:"view"`
	Payload  Expression      `json:"payload,omitempty"`
	Position Position        `json:"position"`
}

func (n *SurfaceStatement) Pos() Position  { return n.Position }
func (n *SurfaceStatement) statementNode() {}

type WaitStatement struct {
	Amount   int64    `json:"amount"`
	Unit     string   `json:"unit,omitempty"`
	Position Position `json:"position"`
}

func (n *WaitStatement) Pos() Position  { return n.Position }
func (n *WaitStatement) statementNode() {}

// CallStatement represents "CALL <tool> WITH <input> AS <alias>" (DSL v1). // CLSF-51
// Input and Alias are optional — a bare CALL <tool> is valid syntax.
type CallStatement struct {
	Tool     *IdentifierExpr `json:"tool"`
	Input    Expression      `json:"input,omitempty"`
	Alias    *IdentifierExpr `json:"alias,omitempty"`
	Position Position        `json:"position"`
}

func (n *CallStatement) Pos() Position  { return n.Position }
func (n *CallStatement) statementNode() {}

// ApproveStatement represents "APPROVE <stage> role <role>" (DSL v1). // CLSF-51
// Role is optional — approval without a specific role targets the workflow owner.
type ApproveStatement struct {
	Stage    *IdentifierExpr `json:"stage"`
	Role     *IdentifierExpr `json:"role,omitempty"`
	Position Position        `json:"position"`
}

func (n *ApproveStatement) Pos() Position  { return n.Position }
func (n *ApproveStatement) statementNode() {}

type IdentifierExpr struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

func (n *IdentifierExpr) Pos() Position   { return n.Position }
func (n *IdentifierExpr) expressionNode() {}

type LiteralExpr struct {
	Value    any      `json:"value"`
	Position Position `json:"position"`
}

func (n *LiteralExpr) Pos() Position   { return n.Position }
func (n *LiteralExpr) expressionNode() {}

type ArrayLiteralExpr struct {
	Elements []Expression `json:"elements,omitempty"`
	Position Position     `json:"position"`
}

func (n *ArrayLiteralExpr) Pos() Position   { return n.Position }
func (n *ArrayLiteralExpr) expressionNode() {}

type ObjectField struct {
	Key      string     `json:"key"`
	Value    Expression `json:"value"`
	Position Position   `json:"position"`
}

type ObjectLiteralExpr struct {
	Fields   []ObjectField `json:"fields,omitempty"`
	Position Position      `json:"position"`
}

func (n *ObjectLiteralExpr) Pos() Position   { return n.Position }
func (n *ObjectLiteralExpr) expressionNode() {}

type ComparisonExpr struct {
	Left     Expression `json:"left"`
	Operator TokenType  `json:"operator"`
	Right    Expression `json:"right"`
	Position Position   `json:"position"`
}

func (n *ComparisonExpr) Pos() Position   { return n.Position }
func (n *ComparisonExpr) expressionNode() {}

func ProgramStatements(program *Program) []Statement {
	if program == nil || program.Workflow == nil {
		return nil
	}
	return program.Workflow.Body
}
