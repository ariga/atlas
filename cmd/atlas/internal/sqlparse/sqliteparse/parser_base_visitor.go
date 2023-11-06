// Code generated from Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package sqliteparse // Parser
import "github.com/antlr4-go/antlr/v4"

type BaseParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseParserVisitor) VisitParse(ctx *ParseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSql_stmt_list(ctx *Sql_stmt_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSql_stmt(ctx *Sql_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAlter_table_stmt(ctx *Alter_table_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAnalyze_stmt(ctx *Analyze_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAttach_stmt(ctx *Attach_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitBegin_stmt(ctx *Begin_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCommit_stmt(ctx *Commit_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitRollback_stmt(ctx *Rollback_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSavepoint_stmt(ctx *Savepoint_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitRelease_stmt(ctx *Release_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCreate_index_stmt(ctx *Create_index_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitIndexed_column(ctx *Indexed_columnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCreate_table_stmt(ctx *Create_table_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitColumn_def(ctx *Column_defContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitType_name(ctx *Type_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitColumn_constraint(ctx *Column_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSigned_number(ctx *Signed_numberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_constraint(ctx *Table_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitForeign_key_clause(ctx *Foreign_key_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitConflict_clause(ctx *Conflict_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCreate_trigger_stmt(ctx *Create_trigger_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCreate_view_stmt(ctx *Create_view_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCreate_virtual_table_stmt(ctx *Create_virtual_table_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitWith_clause(ctx *With_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCte_table_name(ctx *Cte_table_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitRecursive_cte(ctx *Recursive_cteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCommon_table_expression(ctx *Common_table_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitDelete_stmt(ctx *Delete_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitDelete_stmt_limited(ctx *Delete_stmt_limitedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitDetach_stmt(ctx *Detach_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitDrop_stmt(ctx *Drop_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitExpr(ctx *ExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitRaise_function(ctx *Raise_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitLiteral_value(ctx *Literal_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitInsert_stmt(ctx *Insert_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitReturning_clause(ctx *Returning_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitUpsert_clause(ctx *Upsert_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitPragma_stmt(ctx *Pragma_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitPragma_value(ctx *Pragma_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitReindex_stmt(ctx *Reindex_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSelect_stmt(ctx *Select_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitJoin_clause(ctx *Join_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSelect_core(ctx *Select_coreContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFactored_select_stmt(ctx *Factored_select_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSimple_select_stmt(ctx *Simple_select_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCompound_select_stmt(ctx *Compound_select_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_or_subquery(ctx *Table_or_subqueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitResult_column(ctx *Result_columnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitJoin_operator(ctx *Join_operatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitJoin_constraint(ctx *Join_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCompound_operator(ctx *Compound_operatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitUpdate_stmt(ctx *Update_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAssignment_list(ctx *Assignment_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAssignment(ctx *AssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitColumn_name_list(ctx *Column_name_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitUpdate_stmt_limited(ctx *Update_stmt_limitedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitQualified_table_name(ctx *Qualified_table_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitVacuum_stmt(ctx *Vacuum_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFilter_clause(ctx *Filter_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitWindow_defn(ctx *Window_defnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOver_clause(ctx *Over_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFrame_spec(ctx *Frame_specContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFrame_clause(ctx *Frame_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSimple_function_invocation(ctx *Simple_function_invocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAggregate_function_invocation(ctx *Aggregate_function_invocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitWindow_function_invocation(ctx *Window_function_invocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCommon_table_stmt(ctx *Common_table_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOrder_by_stmt(ctx *Order_by_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitLimit_stmt(ctx *Limit_stmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOrdering_term(ctx *Ordering_termContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAsc_desc(ctx *Asc_descContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFrame_left(ctx *Frame_leftContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFrame_right(ctx *Frame_rightContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFrame_single(ctx *Frame_singleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitWindow_function(ctx *Window_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOffset(ctx *OffsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitDefault_value(ctx *Default_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitPartition_by(ctx *Partition_byContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOrder_by_expr(ctx *Order_by_exprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitOrder_by_expr_asc_desc(ctx *Order_by_expr_asc_descContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitExpr_asc_desc(ctx *Expr_asc_descContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitInitial_select(ctx *Initial_selectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitRecursive_select(ctx *Recursive_selectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitUnary_operator(ctx *Unary_operatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitError_message(ctx *Error_messageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitModule_argument(ctx *Module_argumentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitColumn_alias(ctx *Column_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitKeyword(ctx *KeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitName(ctx *NameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFunction_name(ctx *Function_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSchema_name(ctx *Schema_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_name(ctx *Table_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_or_index_name(ctx *Table_or_index_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitColumn_name(ctx *Column_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitCollation_name(ctx *Collation_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitForeign_table(ctx *Foreign_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitIndex_name(ctx *Index_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTrigger_name(ctx *Trigger_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitView_name(ctx *View_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitModule_name(ctx *Module_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitPragma_name(ctx *Pragma_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSavepoint_name(ctx *Savepoint_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_alias(ctx *Table_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTransaction_name(ctx *Transaction_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitWindow_name(ctx *Window_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAlias(ctx *AliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitFilename(ctx *FilenameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitBase_window_name(ctx *Base_window_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitSimple_func(ctx *Simple_funcContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAggregate_func(ctx *Aggregate_funcContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitTable_function_name(ctx *Table_function_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseParserVisitor) VisitAny_name(ctx *Any_nameContext) interface{} {
	return v.VisitChildren(ctx)
}
