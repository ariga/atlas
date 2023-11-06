// Code generated from Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package sqliteparse // Parser
import "github.com/antlr4-go/antlr/v4"

// BaseParserListener is a complete listener for a parse tree produced by Parser.
type BaseParserListener struct{}

var _ ParserListener = &BaseParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterParse is called when production parse is entered.
func (s *BaseParserListener) EnterParse(ctx *ParseContext) {}

// ExitParse is called when production parse is exited.
func (s *BaseParserListener) ExitParse(ctx *ParseContext) {}

// EnterSql_stmt_list is called when production sql_stmt_list is entered.
func (s *BaseParserListener) EnterSql_stmt_list(ctx *Sql_stmt_listContext) {}

// ExitSql_stmt_list is called when production sql_stmt_list is exited.
func (s *BaseParserListener) ExitSql_stmt_list(ctx *Sql_stmt_listContext) {}

// EnterSql_stmt is called when production sql_stmt is entered.
func (s *BaseParserListener) EnterSql_stmt(ctx *Sql_stmtContext) {}

// ExitSql_stmt is called when production sql_stmt is exited.
func (s *BaseParserListener) ExitSql_stmt(ctx *Sql_stmtContext) {}

// EnterAlter_table_stmt is called when production alter_table_stmt is entered.
func (s *BaseParserListener) EnterAlter_table_stmt(ctx *Alter_table_stmtContext) {}

// ExitAlter_table_stmt is called when production alter_table_stmt is exited.
func (s *BaseParserListener) ExitAlter_table_stmt(ctx *Alter_table_stmtContext) {}

// EnterAnalyze_stmt is called when production analyze_stmt is entered.
func (s *BaseParserListener) EnterAnalyze_stmt(ctx *Analyze_stmtContext) {}

// ExitAnalyze_stmt is called when production analyze_stmt is exited.
func (s *BaseParserListener) ExitAnalyze_stmt(ctx *Analyze_stmtContext) {}

// EnterAttach_stmt is called when production attach_stmt is entered.
func (s *BaseParserListener) EnterAttach_stmt(ctx *Attach_stmtContext) {}

// ExitAttach_stmt is called when production attach_stmt is exited.
func (s *BaseParserListener) ExitAttach_stmt(ctx *Attach_stmtContext) {}

// EnterBegin_stmt is called when production begin_stmt is entered.
func (s *BaseParserListener) EnterBegin_stmt(ctx *Begin_stmtContext) {}

// ExitBegin_stmt is called when production begin_stmt is exited.
func (s *BaseParserListener) ExitBegin_stmt(ctx *Begin_stmtContext) {}

// EnterCommit_stmt is called when production commit_stmt is entered.
func (s *BaseParserListener) EnterCommit_stmt(ctx *Commit_stmtContext) {}

// ExitCommit_stmt is called when production commit_stmt is exited.
func (s *BaseParserListener) ExitCommit_stmt(ctx *Commit_stmtContext) {}

// EnterRollback_stmt is called when production rollback_stmt is entered.
func (s *BaseParserListener) EnterRollback_stmt(ctx *Rollback_stmtContext) {}

// ExitRollback_stmt is called when production rollback_stmt is exited.
func (s *BaseParserListener) ExitRollback_stmt(ctx *Rollback_stmtContext) {}

// EnterSavepoint_stmt is called when production savepoint_stmt is entered.
func (s *BaseParserListener) EnterSavepoint_stmt(ctx *Savepoint_stmtContext) {}

// ExitSavepoint_stmt is called when production savepoint_stmt is exited.
func (s *BaseParserListener) ExitSavepoint_stmt(ctx *Savepoint_stmtContext) {}

// EnterRelease_stmt is called when production release_stmt is entered.
func (s *BaseParserListener) EnterRelease_stmt(ctx *Release_stmtContext) {}

// ExitRelease_stmt is called when production release_stmt is exited.
func (s *BaseParserListener) ExitRelease_stmt(ctx *Release_stmtContext) {}

// EnterCreate_index_stmt is called when production create_index_stmt is entered.
func (s *BaseParserListener) EnterCreate_index_stmt(ctx *Create_index_stmtContext) {}

// ExitCreate_index_stmt is called when production create_index_stmt is exited.
func (s *BaseParserListener) ExitCreate_index_stmt(ctx *Create_index_stmtContext) {}

// EnterIndexed_column is called when production indexed_column is entered.
func (s *BaseParserListener) EnterIndexed_column(ctx *Indexed_columnContext) {}

// ExitIndexed_column is called when production indexed_column is exited.
func (s *BaseParserListener) ExitIndexed_column(ctx *Indexed_columnContext) {}

// EnterCreate_table_stmt is called when production create_table_stmt is entered.
func (s *BaseParserListener) EnterCreate_table_stmt(ctx *Create_table_stmtContext) {}

// ExitCreate_table_stmt is called when production create_table_stmt is exited.
func (s *BaseParserListener) ExitCreate_table_stmt(ctx *Create_table_stmtContext) {}

// EnterColumn_def is called when production column_def is entered.
func (s *BaseParserListener) EnterColumn_def(ctx *Column_defContext) {}

// ExitColumn_def is called when production column_def is exited.
func (s *BaseParserListener) ExitColumn_def(ctx *Column_defContext) {}

// EnterType_name is called when production type_name is entered.
func (s *BaseParserListener) EnterType_name(ctx *Type_nameContext) {}

// ExitType_name is called when production type_name is exited.
func (s *BaseParserListener) ExitType_name(ctx *Type_nameContext) {}

// EnterColumn_constraint is called when production column_constraint is entered.
func (s *BaseParserListener) EnterColumn_constraint(ctx *Column_constraintContext) {}

// ExitColumn_constraint is called when production column_constraint is exited.
func (s *BaseParserListener) ExitColumn_constraint(ctx *Column_constraintContext) {}

// EnterSigned_number is called when production signed_number is entered.
func (s *BaseParserListener) EnterSigned_number(ctx *Signed_numberContext) {}

// ExitSigned_number is called when production signed_number is exited.
func (s *BaseParserListener) ExitSigned_number(ctx *Signed_numberContext) {}

// EnterTable_constraint is called when production table_constraint is entered.
func (s *BaseParserListener) EnterTable_constraint(ctx *Table_constraintContext) {}

// ExitTable_constraint is called when production table_constraint is exited.
func (s *BaseParserListener) ExitTable_constraint(ctx *Table_constraintContext) {}

// EnterForeign_key_clause is called when production foreign_key_clause is entered.
func (s *BaseParserListener) EnterForeign_key_clause(ctx *Foreign_key_clauseContext) {}

// ExitForeign_key_clause is called when production foreign_key_clause is exited.
func (s *BaseParserListener) ExitForeign_key_clause(ctx *Foreign_key_clauseContext) {}

// EnterConflict_clause is called when production conflict_clause is entered.
func (s *BaseParserListener) EnterConflict_clause(ctx *Conflict_clauseContext) {}

// ExitConflict_clause is called when production conflict_clause is exited.
func (s *BaseParserListener) ExitConflict_clause(ctx *Conflict_clauseContext) {}

// EnterCreate_trigger_stmt is called when production create_trigger_stmt is entered.
func (s *BaseParserListener) EnterCreate_trigger_stmt(ctx *Create_trigger_stmtContext) {}

// ExitCreate_trigger_stmt is called when production create_trigger_stmt is exited.
func (s *BaseParserListener) ExitCreate_trigger_stmt(ctx *Create_trigger_stmtContext) {}

// EnterCreate_view_stmt is called when production create_view_stmt is entered.
func (s *BaseParserListener) EnterCreate_view_stmt(ctx *Create_view_stmtContext) {}

// ExitCreate_view_stmt is called when production create_view_stmt is exited.
func (s *BaseParserListener) ExitCreate_view_stmt(ctx *Create_view_stmtContext) {}

// EnterCreate_virtual_table_stmt is called when production create_virtual_table_stmt is entered.
func (s *BaseParserListener) EnterCreate_virtual_table_stmt(ctx *Create_virtual_table_stmtContext) {}

// ExitCreate_virtual_table_stmt is called when production create_virtual_table_stmt is exited.
func (s *BaseParserListener) ExitCreate_virtual_table_stmt(ctx *Create_virtual_table_stmtContext) {}

// EnterWith_clause is called when production with_clause is entered.
func (s *BaseParserListener) EnterWith_clause(ctx *With_clauseContext) {}

// ExitWith_clause is called when production with_clause is exited.
func (s *BaseParserListener) ExitWith_clause(ctx *With_clauseContext) {}

// EnterCte_table_name is called when production cte_table_name is entered.
func (s *BaseParserListener) EnterCte_table_name(ctx *Cte_table_nameContext) {}

// ExitCte_table_name is called when production cte_table_name is exited.
func (s *BaseParserListener) ExitCte_table_name(ctx *Cte_table_nameContext) {}

// EnterRecursive_cte is called when production recursive_cte is entered.
func (s *BaseParserListener) EnterRecursive_cte(ctx *Recursive_cteContext) {}

// ExitRecursive_cte is called when production recursive_cte is exited.
func (s *BaseParserListener) ExitRecursive_cte(ctx *Recursive_cteContext) {}

// EnterCommon_table_expression is called when production common_table_expression is entered.
func (s *BaseParserListener) EnterCommon_table_expression(ctx *Common_table_expressionContext) {}

// ExitCommon_table_expression is called when production common_table_expression is exited.
func (s *BaseParserListener) ExitCommon_table_expression(ctx *Common_table_expressionContext) {}

// EnterDelete_stmt is called when production delete_stmt is entered.
func (s *BaseParserListener) EnterDelete_stmt(ctx *Delete_stmtContext) {}

// ExitDelete_stmt is called when production delete_stmt is exited.
func (s *BaseParserListener) ExitDelete_stmt(ctx *Delete_stmtContext) {}

// EnterDelete_stmt_limited is called when production delete_stmt_limited is entered.
func (s *BaseParserListener) EnterDelete_stmt_limited(ctx *Delete_stmt_limitedContext) {}

// ExitDelete_stmt_limited is called when production delete_stmt_limited is exited.
func (s *BaseParserListener) ExitDelete_stmt_limited(ctx *Delete_stmt_limitedContext) {}

// EnterDetach_stmt is called when production detach_stmt is entered.
func (s *BaseParserListener) EnterDetach_stmt(ctx *Detach_stmtContext) {}

// ExitDetach_stmt is called when production detach_stmt is exited.
func (s *BaseParserListener) ExitDetach_stmt(ctx *Detach_stmtContext) {}

// EnterDrop_stmt is called when production drop_stmt is entered.
func (s *BaseParserListener) EnterDrop_stmt(ctx *Drop_stmtContext) {}

// ExitDrop_stmt is called when production drop_stmt is exited.
func (s *BaseParserListener) ExitDrop_stmt(ctx *Drop_stmtContext) {}

// EnterExpr is called when production expr is entered.
func (s *BaseParserListener) EnterExpr(ctx *ExprContext) {}

// ExitExpr is called when production expr is exited.
func (s *BaseParserListener) ExitExpr(ctx *ExprContext) {}

// EnterRaise_function is called when production raise_function is entered.
func (s *BaseParserListener) EnterRaise_function(ctx *Raise_functionContext) {}

// ExitRaise_function is called when production raise_function is exited.
func (s *BaseParserListener) ExitRaise_function(ctx *Raise_functionContext) {}

// EnterLiteral_value is called when production literal_value is entered.
func (s *BaseParserListener) EnterLiteral_value(ctx *Literal_valueContext) {}

// ExitLiteral_value is called when production literal_value is exited.
func (s *BaseParserListener) ExitLiteral_value(ctx *Literal_valueContext) {}

// EnterInsert_stmt is called when production insert_stmt is entered.
func (s *BaseParserListener) EnterInsert_stmt(ctx *Insert_stmtContext) {}

// ExitInsert_stmt is called when production insert_stmt is exited.
func (s *BaseParserListener) ExitInsert_stmt(ctx *Insert_stmtContext) {}

// EnterReturning_clause is called when production returning_clause is entered.
func (s *BaseParserListener) EnterReturning_clause(ctx *Returning_clauseContext) {}

// ExitReturning_clause is called when production returning_clause is exited.
func (s *BaseParserListener) ExitReturning_clause(ctx *Returning_clauseContext) {}

// EnterUpsert_clause is called when production upsert_clause is entered.
func (s *BaseParserListener) EnterUpsert_clause(ctx *Upsert_clauseContext) {}

// ExitUpsert_clause is called when production upsert_clause is exited.
func (s *BaseParserListener) ExitUpsert_clause(ctx *Upsert_clauseContext) {}

// EnterPragma_stmt is called when production pragma_stmt is entered.
func (s *BaseParserListener) EnterPragma_stmt(ctx *Pragma_stmtContext) {}

// ExitPragma_stmt is called when production pragma_stmt is exited.
func (s *BaseParserListener) ExitPragma_stmt(ctx *Pragma_stmtContext) {}

// EnterPragma_value is called when production pragma_value is entered.
func (s *BaseParserListener) EnterPragma_value(ctx *Pragma_valueContext) {}

// ExitPragma_value is called when production pragma_value is exited.
func (s *BaseParserListener) ExitPragma_value(ctx *Pragma_valueContext) {}

// EnterReindex_stmt is called when production reindex_stmt is entered.
func (s *BaseParserListener) EnterReindex_stmt(ctx *Reindex_stmtContext) {}

// ExitReindex_stmt is called when production reindex_stmt is exited.
func (s *BaseParserListener) ExitReindex_stmt(ctx *Reindex_stmtContext) {}

// EnterSelect_stmt is called when production select_stmt is entered.
func (s *BaseParserListener) EnterSelect_stmt(ctx *Select_stmtContext) {}

// ExitSelect_stmt is called when production select_stmt is exited.
func (s *BaseParserListener) ExitSelect_stmt(ctx *Select_stmtContext) {}

// EnterJoin_clause is called when production join_clause is entered.
func (s *BaseParserListener) EnterJoin_clause(ctx *Join_clauseContext) {}

// ExitJoin_clause is called when production join_clause is exited.
func (s *BaseParserListener) ExitJoin_clause(ctx *Join_clauseContext) {}

// EnterSelect_core is called when production select_core is entered.
func (s *BaseParserListener) EnterSelect_core(ctx *Select_coreContext) {}

// ExitSelect_core is called when production select_core is exited.
func (s *BaseParserListener) ExitSelect_core(ctx *Select_coreContext) {}

// EnterFactored_select_stmt is called when production factored_select_stmt is entered.
func (s *BaseParserListener) EnterFactored_select_stmt(ctx *Factored_select_stmtContext) {}

// ExitFactored_select_stmt is called when production factored_select_stmt is exited.
func (s *BaseParserListener) ExitFactored_select_stmt(ctx *Factored_select_stmtContext) {}

// EnterSimple_select_stmt is called when production simple_select_stmt is entered.
func (s *BaseParserListener) EnterSimple_select_stmt(ctx *Simple_select_stmtContext) {}

// ExitSimple_select_stmt is called when production simple_select_stmt is exited.
func (s *BaseParserListener) ExitSimple_select_stmt(ctx *Simple_select_stmtContext) {}

// EnterCompound_select_stmt is called when production compound_select_stmt is entered.
func (s *BaseParserListener) EnterCompound_select_stmt(ctx *Compound_select_stmtContext) {}

// ExitCompound_select_stmt is called when production compound_select_stmt is exited.
func (s *BaseParserListener) ExitCompound_select_stmt(ctx *Compound_select_stmtContext) {}

// EnterTable_or_subquery is called when production table_or_subquery is entered.
func (s *BaseParserListener) EnterTable_or_subquery(ctx *Table_or_subqueryContext) {}

// ExitTable_or_subquery is called when production table_or_subquery is exited.
func (s *BaseParserListener) ExitTable_or_subquery(ctx *Table_or_subqueryContext) {}

// EnterResult_column is called when production result_column is entered.
func (s *BaseParserListener) EnterResult_column(ctx *Result_columnContext) {}

// ExitResult_column is called when production result_column is exited.
func (s *BaseParserListener) ExitResult_column(ctx *Result_columnContext) {}

// EnterJoin_operator is called when production join_operator is entered.
func (s *BaseParserListener) EnterJoin_operator(ctx *Join_operatorContext) {}

// ExitJoin_operator is called when production join_operator is exited.
func (s *BaseParserListener) ExitJoin_operator(ctx *Join_operatorContext) {}

// EnterJoin_constraint is called when production join_constraint is entered.
func (s *BaseParserListener) EnterJoin_constraint(ctx *Join_constraintContext) {}

// ExitJoin_constraint is called when production join_constraint is exited.
func (s *BaseParserListener) ExitJoin_constraint(ctx *Join_constraintContext) {}

// EnterCompound_operator is called when production compound_operator is entered.
func (s *BaseParserListener) EnterCompound_operator(ctx *Compound_operatorContext) {}

// ExitCompound_operator is called when production compound_operator is exited.
func (s *BaseParserListener) ExitCompound_operator(ctx *Compound_operatorContext) {}

// EnterUpdate_stmt is called when production update_stmt is entered.
func (s *BaseParserListener) EnterUpdate_stmt(ctx *Update_stmtContext) {}

// ExitUpdate_stmt is called when production update_stmt is exited.
func (s *BaseParserListener) ExitUpdate_stmt(ctx *Update_stmtContext) {}

// EnterAssignment_list is called when production assignment_list is entered.
func (s *BaseParserListener) EnterAssignment_list(ctx *Assignment_listContext) {}

// ExitAssignment_list is called when production assignment_list is exited.
func (s *BaseParserListener) ExitAssignment_list(ctx *Assignment_listContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseParserListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseParserListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterColumn_name_list is called when production column_name_list is entered.
func (s *BaseParserListener) EnterColumn_name_list(ctx *Column_name_listContext) {}

// ExitColumn_name_list is called when production column_name_list is exited.
func (s *BaseParserListener) ExitColumn_name_list(ctx *Column_name_listContext) {}

// EnterUpdate_stmt_limited is called when production update_stmt_limited is entered.
func (s *BaseParserListener) EnterUpdate_stmt_limited(ctx *Update_stmt_limitedContext) {}

// ExitUpdate_stmt_limited is called when production update_stmt_limited is exited.
func (s *BaseParserListener) ExitUpdate_stmt_limited(ctx *Update_stmt_limitedContext) {}

// EnterQualified_table_name is called when production qualified_table_name is entered.
func (s *BaseParserListener) EnterQualified_table_name(ctx *Qualified_table_nameContext) {}

// ExitQualified_table_name is called when production qualified_table_name is exited.
func (s *BaseParserListener) ExitQualified_table_name(ctx *Qualified_table_nameContext) {}

// EnterVacuum_stmt is called when production vacuum_stmt is entered.
func (s *BaseParserListener) EnterVacuum_stmt(ctx *Vacuum_stmtContext) {}

// ExitVacuum_stmt is called when production vacuum_stmt is exited.
func (s *BaseParserListener) ExitVacuum_stmt(ctx *Vacuum_stmtContext) {}

// EnterFilter_clause is called when production filter_clause is entered.
func (s *BaseParserListener) EnterFilter_clause(ctx *Filter_clauseContext) {}

// ExitFilter_clause is called when production filter_clause is exited.
func (s *BaseParserListener) ExitFilter_clause(ctx *Filter_clauseContext) {}

// EnterWindow_defn is called when production window_defn is entered.
func (s *BaseParserListener) EnterWindow_defn(ctx *Window_defnContext) {}

// ExitWindow_defn is called when production window_defn is exited.
func (s *BaseParserListener) ExitWindow_defn(ctx *Window_defnContext) {}

// EnterOver_clause is called when production over_clause is entered.
func (s *BaseParserListener) EnterOver_clause(ctx *Over_clauseContext) {}

// ExitOver_clause is called when production over_clause is exited.
func (s *BaseParserListener) ExitOver_clause(ctx *Over_clauseContext) {}

// EnterFrame_spec is called when production frame_spec is entered.
func (s *BaseParserListener) EnterFrame_spec(ctx *Frame_specContext) {}

// ExitFrame_spec is called when production frame_spec is exited.
func (s *BaseParserListener) ExitFrame_spec(ctx *Frame_specContext) {}

// EnterFrame_clause is called when production frame_clause is entered.
func (s *BaseParserListener) EnterFrame_clause(ctx *Frame_clauseContext) {}

// ExitFrame_clause is called when production frame_clause is exited.
func (s *BaseParserListener) ExitFrame_clause(ctx *Frame_clauseContext) {}

// EnterSimple_function_invocation is called when production simple_function_invocation is entered.
func (s *BaseParserListener) EnterSimple_function_invocation(ctx *Simple_function_invocationContext) {
}

// ExitSimple_function_invocation is called when production simple_function_invocation is exited.
func (s *BaseParserListener) ExitSimple_function_invocation(ctx *Simple_function_invocationContext) {}

// EnterAggregate_function_invocation is called when production aggregate_function_invocation is entered.
func (s *BaseParserListener) EnterAggregate_function_invocation(ctx *Aggregate_function_invocationContext) {
}

// ExitAggregate_function_invocation is called when production aggregate_function_invocation is exited.
func (s *BaseParserListener) ExitAggregate_function_invocation(ctx *Aggregate_function_invocationContext) {
}

// EnterWindow_function_invocation is called when production window_function_invocation is entered.
func (s *BaseParserListener) EnterWindow_function_invocation(ctx *Window_function_invocationContext) {
}

// ExitWindow_function_invocation is called when production window_function_invocation is exited.
func (s *BaseParserListener) ExitWindow_function_invocation(ctx *Window_function_invocationContext) {}

// EnterCommon_table_stmt is called when production common_table_stmt is entered.
func (s *BaseParserListener) EnterCommon_table_stmt(ctx *Common_table_stmtContext) {}

// ExitCommon_table_stmt is called when production common_table_stmt is exited.
func (s *BaseParserListener) ExitCommon_table_stmt(ctx *Common_table_stmtContext) {}

// EnterOrder_by_stmt is called when production order_by_stmt is entered.
func (s *BaseParserListener) EnterOrder_by_stmt(ctx *Order_by_stmtContext) {}

// ExitOrder_by_stmt is called when production order_by_stmt is exited.
func (s *BaseParserListener) ExitOrder_by_stmt(ctx *Order_by_stmtContext) {}

// EnterLimit_stmt is called when production limit_stmt is entered.
func (s *BaseParserListener) EnterLimit_stmt(ctx *Limit_stmtContext) {}

// ExitLimit_stmt is called when production limit_stmt is exited.
func (s *BaseParserListener) ExitLimit_stmt(ctx *Limit_stmtContext) {}

// EnterOrdering_term is called when production ordering_term is entered.
func (s *BaseParserListener) EnterOrdering_term(ctx *Ordering_termContext) {}

// ExitOrdering_term is called when production ordering_term is exited.
func (s *BaseParserListener) ExitOrdering_term(ctx *Ordering_termContext) {}

// EnterAsc_desc is called when production asc_desc is entered.
func (s *BaseParserListener) EnterAsc_desc(ctx *Asc_descContext) {}

// ExitAsc_desc is called when production asc_desc is exited.
func (s *BaseParserListener) ExitAsc_desc(ctx *Asc_descContext) {}

// EnterFrame_left is called when production frame_left is entered.
func (s *BaseParserListener) EnterFrame_left(ctx *Frame_leftContext) {}

// ExitFrame_left is called when production frame_left is exited.
func (s *BaseParserListener) ExitFrame_left(ctx *Frame_leftContext) {}

// EnterFrame_right is called when production frame_right is entered.
func (s *BaseParserListener) EnterFrame_right(ctx *Frame_rightContext) {}

// ExitFrame_right is called when production frame_right is exited.
func (s *BaseParserListener) ExitFrame_right(ctx *Frame_rightContext) {}

// EnterFrame_single is called when production frame_single is entered.
func (s *BaseParserListener) EnterFrame_single(ctx *Frame_singleContext) {}

// ExitFrame_single is called when production frame_single is exited.
func (s *BaseParserListener) ExitFrame_single(ctx *Frame_singleContext) {}

// EnterWindow_function is called when production window_function is entered.
func (s *BaseParserListener) EnterWindow_function(ctx *Window_functionContext) {}

// ExitWindow_function is called when production window_function is exited.
func (s *BaseParserListener) ExitWindow_function(ctx *Window_functionContext) {}

// EnterOffset is called when production offset is entered.
func (s *BaseParserListener) EnterOffset(ctx *OffsetContext) {}

// ExitOffset is called when production offset is exited.
func (s *BaseParserListener) ExitOffset(ctx *OffsetContext) {}

// EnterDefault_value is called when production default_value is entered.
func (s *BaseParserListener) EnterDefault_value(ctx *Default_valueContext) {}

// ExitDefault_value is called when production default_value is exited.
func (s *BaseParserListener) ExitDefault_value(ctx *Default_valueContext) {}

// EnterPartition_by is called when production partition_by is entered.
func (s *BaseParserListener) EnterPartition_by(ctx *Partition_byContext) {}

// ExitPartition_by is called when production partition_by is exited.
func (s *BaseParserListener) ExitPartition_by(ctx *Partition_byContext) {}

// EnterOrder_by_expr is called when production order_by_expr is entered.
func (s *BaseParserListener) EnterOrder_by_expr(ctx *Order_by_exprContext) {}

// ExitOrder_by_expr is called when production order_by_expr is exited.
func (s *BaseParserListener) ExitOrder_by_expr(ctx *Order_by_exprContext) {}

// EnterOrder_by_expr_asc_desc is called when production order_by_expr_asc_desc is entered.
func (s *BaseParserListener) EnterOrder_by_expr_asc_desc(ctx *Order_by_expr_asc_descContext) {}

// ExitOrder_by_expr_asc_desc is called when production order_by_expr_asc_desc is exited.
func (s *BaseParserListener) ExitOrder_by_expr_asc_desc(ctx *Order_by_expr_asc_descContext) {}

// EnterExpr_asc_desc is called when production expr_asc_desc is entered.
func (s *BaseParserListener) EnterExpr_asc_desc(ctx *Expr_asc_descContext) {}

// ExitExpr_asc_desc is called when production expr_asc_desc is exited.
func (s *BaseParserListener) ExitExpr_asc_desc(ctx *Expr_asc_descContext) {}

// EnterInitial_select is called when production initial_select is entered.
func (s *BaseParserListener) EnterInitial_select(ctx *Initial_selectContext) {}

// ExitInitial_select is called when production initial_select is exited.
func (s *BaseParserListener) ExitInitial_select(ctx *Initial_selectContext) {}

// EnterRecursive_select is called when production recursive_select is entered.
func (s *BaseParserListener) EnterRecursive_select(ctx *Recursive_selectContext) {}

// ExitRecursive_select is called when production recursive_select is exited.
func (s *BaseParserListener) ExitRecursive_select(ctx *Recursive_selectContext) {}

// EnterUnary_operator is called when production unary_operator is entered.
func (s *BaseParserListener) EnterUnary_operator(ctx *Unary_operatorContext) {}

// ExitUnary_operator is called when production unary_operator is exited.
func (s *BaseParserListener) ExitUnary_operator(ctx *Unary_operatorContext) {}

// EnterError_message is called when production error_message is entered.
func (s *BaseParserListener) EnterError_message(ctx *Error_messageContext) {}

// ExitError_message is called when production error_message is exited.
func (s *BaseParserListener) ExitError_message(ctx *Error_messageContext) {}

// EnterModule_argument is called when production module_argument is entered.
func (s *BaseParserListener) EnterModule_argument(ctx *Module_argumentContext) {}

// ExitModule_argument is called when production module_argument is exited.
func (s *BaseParserListener) ExitModule_argument(ctx *Module_argumentContext) {}

// EnterColumn_alias is called when production column_alias is entered.
func (s *BaseParserListener) EnterColumn_alias(ctx *Column_aliasContext) {}

// ExitColumn_alias is called when production column_alias is exited.
func (s *BaseParserListener) ExitColumn_alias(ctx *Column_aliasContext) {}

// EnterKeyword is called when production keyword is entered.
func (s *BaseParserListener) EnterKeyword(ctx *KeywordContext) {}

// ExitKeyword is called when production keyword is exited.
func (s *BaseParserListener) ExitKeyword(ctx *KeywordContext) {}

// EnterName is called when production name is entered.
func (s *BaseParserListener) EnterName(ctx *NameContext) {}

// ExitName is called when production name is exited.
func (s *BaseParserListener) ExitName(ctx *NameContext) {}

// EnterFunction_name is called when production function_name is entered.
func (s *BaseParserListener) EnterFunction_name(ctx *Function_nameContext) {}

// ExitFunction_name is called when production function_name is exited.
func (s *BaseParserListener) ExitFunction_name(ctx *Function_nameContext) {}

// EnterSchema_name is called when production schema_name is entered.
func (s *BaseParserListener) EnterSchema_name(ctx *Schema_nameContext) {}

// ExitSchema_name is called when production schema_name is exited.
func (s *BaseParserListener) ExitSchema_name(ctx *Schema_nameContext) {}

// EnterTable_name is called when production table_name is entered.
func (s *BaseParserListener) EnterTable_name(ctx *Table_nameContext) {}

// ExitTable_name is called when production table_name is exited.
func (s *BaseParserListener) ExitTable_name(ctx *Table_nameContext) {}

// EnterTable_or_index_name is called when production table_or_index_name is entered.
func (s *BaseParserListener) EnterTable_or_index_name(ctx *Table_or_index_nameContext) {}

// ExitTable_or_index_name is called when production table_or_index_name is exited.
func (s *BaseParserListener) ExitTable_or_index_name(ctx *Table_or_index_nameContext) {}

// EnterColumn_name is called when production column_name is entered.
func (s *BaseParserListener) EnterColumn_name(ctx *Column_nameContext) {}

// ExitColumn_name is called when production column_name is exited.
func (s *BaseParserListener) ExitColumn_name(ctx *Column_nameContext) {}

// EnterCollation_name is called when production collation_name is entered.
func (s *BaseParserListener) EnterCollation_name(ctx *Collation_nameContext) {}

// ExitCollation_name is called when production collation_name is exited.
func (s *BaseParserListener) ExitCollation_name(ctx *Collation_nameContext) {}

// EnterForeign_table is called when production foreign_table is entered.
func (s *BaseParserListener) EnterForeign_table(ctx *Foreign_tableContext) {}

// ExitForeign_table is called when production foreign_table is exited.
func (s *BaseParserListener) ExitForeign_table(ctx *Foreign_tableContext) {}

// EnterIndex_name is called when production index_name is entered.
func (s *BaseParserListener) EnterIndex_name(ctx *Index_nameContext) {}

// ExitIndex_name is called when production index_name is exited.
func (s *BaseParserListener) ExitIndex_name(ctx *Index_nameContext) {}

// EnterTrigger_name is called when production trigger_name is entered.
func (s *BaseParserListener) EnterTrigger_name(ctx *Trigger_nameContext) {}

// ExitTrigger_name is called when production trigger_name is exited.
func (s *BaseParserListener) ExitTrigger_name(ctx *Trigger_nameContext) {}

// EnterView_name is called when production view_name is entered.
func (s *BaseParserListener) EnterView_name(ctx *View_nameContext) {}

// ExitView_name is called when production view_name is exited.
func (s *BaseParserListener) ExitView_name(ctx *View_nameContext) {}

// EnterModule_name is called when production module_name is entered.
func (s *BaseParserListener) EnterModule_name(ctx *Module_nameContext) {}

// ExitModule_name is called when production module_name is exited.
func (s *BaseParserListener) ExitModule_name(ctx *Module_nameContext) {}

// EnterPragma_name is called when production pragma_name is entered.
func (s *BaseParserListener) EnterPragma_name(ctx *Pragma_nameContext) {}

// ExitPragma_name is called when production pragma_name is exited.
func (s *BaseParserListener) ExitPragma_name(ctx *Pragma_nameContext) {}

// EnterSavepoint_name is called when production savepoint_name is entered.
func (s *BaseParserListener) EnterSavepoint_name(ctx *Savepoint_nameContext) {}

// ExitSavepoint_name is called when production savepoint_name is exited.
func (s *BaseParserListener) ExitSavepoint_name(ctx *Savepoint_nameContext) {}

// EnterTable_alias is called when production table_alias is entered.
func (s *BaseParserListener) EnterTable_alias(ctx *Table_aliasContext) {}

// ExitTable_alias is called when production table_alias is exited.
func (s *BaseParserListener) ExitTable_alias(ctx *Table_aliasContext) {}

// EnterTransaction_name is called when production transaction_name is entered.
func (s *BaseParserListener) EnterTransaction_name(ctx *Transaction_nameContext) {}

// ExitTransaction_name is called when production transaction_name is exited.
func (s *BaseParserListener) ExitTransaction_name(ctx *Transaction_nameContext) {}

// EnterWindow_name is called when production window_name is entered.
func (s *BaseParserListener) EnterWindow_name(ctx *Window_nameContext) {}

// ExitWindow_name is called when production window_name is exited.
func (s *BaseParserListener) ExitWindow_name(ctx *Window_nameContext) {}

// EnterAlias is called when production alias is entered.
func (s *BaseParserListener) EnterAlias(ctx *AliasContext) {}

// ExitAlias is called when production alias is exited.
func (s *BaseParserListener) ExitAlias(ctx *AliasContext) {}

// EnterFilename is called when production filename is entered.
func (s *BaseParserListener) EnterFilename(ctx *FilenameContext) {}

// ExitFilename is called when production filename is exited.
func (s *BaseParserListener) ExitFilename(ctx *FilenameContext) {}

// EnterBase_window_name is called when production base_window_name is entered.
func (s *BaseParserListener) EnterBase_window_name(ctx *Base_window_nameContext) {}

// ExitBase_window_name is called when production base_window_name is exited.
func (s *BaseParserListener) ExitBase_window_name(ctx *Base_window_nameContext) {}

// EnterSimple_func is called when production simple_func is entered.
func (s *BaseParserListener) EnterSimple_func(ctx *Simple_funcContext) {}

// ExitSimple_func is called when production simple_func is exited.
func (s *BaseParserListener) ExitSimple_func(ctx *Simple_funcContext) {}

// EnterAggregate_func is called when production aggregate_func is entered.
func (s *BaseParserListener) EnterAggregate_func(ctx *Aggregate_funcContext) {}

// ExitAggregate_func is called when production aggregate_func is exited.
func (s *BaseParserListener) ExitAggregate_func(ctx *Aggregate_funcContext) {}

// EnterTable_function_name is called when production table_function_name is entered.
func (s *BaseParserListener) EnterTable_function_name(ctx *Table_function_nameContext) {}

// ExitTable_function_name is called when production table_function_name is exited.
func (s *BaseParserListener) ExitTable_function_name(ctx *Table_function_nameContext) {}

// EnterAny_name is called when production any_name is entered.
func (s *BaseParserListener) EnterAny_name(ctx *Any_nameContext) {}

// ExitAny_name is called when production any_name is exited.
func (s *BaseParserListener) ExitAny_name(ctx *Any_nameContext) {}
