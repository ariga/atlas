// Code generated from Parser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package sqliteparse // Parser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by Parser.
type ParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by Parser#parse.
	VisitParse(ctx *ParseContext) interface{}

	// Visit a parse tree produced by Parser#sql_stmt_list.
	VisitSql_stmt_list(ctx *Sql_stmt_listContext) interface{}

	// Visit a parse tree produced by Parser#sql_stmt.
	VisitSql_stmt(ctx *Sql_stmtContext) interface{}

	// Visit a parse tree produced by Parser#alter_table_stmt.
	VisitAlter_table_stmt(ctx *Alter_table_stmtContext) interface{}

	// Visit a parse tree produced by Parser#analyze_stmt.
	VisitAnalyze_stmt(ctx *Analyze_stmtContext) interface{}

	// Visit a parse tree produced by Parser#attach_stmt.
	VisitAttach_stmt(ctx *Attach_stmtContext) interface{}

	// Visit a parse tree produced by Parser#begin_stmt.
	VisitBegin_stmt(ctx *Begin_stmtContext) interface{}

	// Visit a parse tree produced by Parser#commit_stmt.
	VisitCommit_stmt(ctx *Commit_stmtContext) interface{}

	// Visit a parse tree produced by Parser#rollback_stmt.
	VisitRollback_stmt(ctx *Rollback_stmtContext) interface{}

	// Visit a parse tree produced by Parser#savepoint_stmt.
	VisitSavepoint_stmt(ctx *Savepoint_stmtContext) interface{}

	// Visit a parse tree produced by Parser#release_stmt.
	VisitRelease_stmt(ctx *Release_stmtContext) interface{}

	// Visit a parse tree produced by Parser#create_index_stmt.
	VisitCreate_index_stmt(ctx *Create_index_stmtContext) interface{}

	// Visit a parse tree produced by Parser#indexed_column.
	VisitIndexed_column(ctx *Indexed_columnContext) interface{}

	// Visit a parse tree produced by Parser#create_table_stmt.
	VisitCreate_table_stmt(ctx *Create_table_stmtContext) interface{}

	// Visit a parse tree produced by Parser#column_def.
	VisitColumn_def(ctx *Column_defContext) interface{}

	// Visit a parse tree produced by Parser#type_name.
	VisitType_name(ctx *Type_nameContext) interface{}

	// Visit a parse tree produced by Parser#column_constraint.
	VisitColumn_constraint(ctx *Column_constraintContext) interface{}

	// Visit a parse tree produced by Parser#signed_number.
	VisitSigned_number(ctx *Signed_numberContext) interface{}

	// Visit a parse tree produced by Parser#table_constraint.
	VisitTable_constraint(ctx *Table_constraintContext) interface{}

	// Visit a parse tree produced by Parser#foreign_key_clause.
	VisitForeign_key_clause(ctx *Foreign_key_clauseContext) interface{}

	// Visit a parse tree produced by Parser#conflict_clause.
	VisitConflict_clause(ctx *Conflict_clauseContext) interface{}

	// Visit a parse tree produced by Parser#create_trigger_stmt.
	VisitCreate_trigger_stmt(ctx *Create_trigger_stmtContext) interface{}

	// Visit a parse tree produced by Parser#create_view_stmt.
	VisitCreate_view_stmt(ctx *Create_view_stmtContext) interface{}

	// Visit a parse tree produced by Parser#create_virtual_table_stmt.
	VisitCreate_virtual_table_stmt(ctx *Create_virtual_table_stmtContext) interface{}

	// Visit a parse tree produced by Parser#with_clause.
	VisitWith_clause(ctx *With_clauseContext) interface{}

	// Visit a parse tree produced by Parser#cte_table_name.
	VisitCte_table_name(ctx *Cte_table_nameContext) interface{}

	// Visit a parse tree produced by Parser#recursive_cte.
	VisitRecursive_cte(ctx *Recursive_cteContext) interface{}

	// Visit a parse tree produced by Parser#common_table_expression.
	VisitCommon_table_expression(ctx *Common_table_expressionContext) interface{}

	// Visit a parse tree produced by Parser#delete_stmt.
	VisitDelete_stmt(ctx *Delete_stmtContext) interface{}

	// Visit a parse tree produced by Parser#delete_stmt_limited.
	VisitDelete_stmt_limited(ctx *Delete_stmt_limitedContext) interface{}

	// Visit a parse tree produced by Parser#detach_stmt.
	VisitDetach_stmt(ctx *Detach_stmtContext) interface{}

	// Visit a parse tree produced by Parser#drop_stmt.
	VisitDrop_stmt(ctx *Drop_stmtContext) interface{}

	// Visit a parse tree produced by Parser#expr.
	VisitExpr(ctx *ExprContext) interface{}

	// Visit a parse tree produced by Parser#raise_function.
	VisitRaise_function(ctx *Raise_functionContext) interface{}

	// Visit a parse tree produced by Parser#literal_value.
	VisitLiteral_value(ctx *Literal_valueContext) interface{}

	// Visit a parse tree produced by Parser#insert_stmt.
	VisitInsert_stmt(ctx *Insert_stmtContext) interface{}

	// Visit a parse tree produced by Parser#returning_clause.
	VisitReturning_clause(ctx *Returning_clauseContext) interface{}

	// Visit a parse tree produced by Parser#upsert_clause.
	VisitUpsert_clause(ctx *Upsert_clauseContext) interface{}

	// Visit a parse tree produced by Parser#pragma_stmt.
	VisitPragma_stmt(ctx *Pragma_stmtContext) interface{}

	// Visit a parse tree produced by Parser#pragma_value.
	VisitPragma_value(ctx *Pragma_valueContext) interface{}

	// Visit a parse tree produced by Parser#reindex_stmt.
	VisitReindex_stmt(ctx *Reindex_stmtContext) interface{}

	// Visit a parse tree produced by Parser#select_stmt.
	VisitSelect_stmt(ctx *Select_stmtContext) interface{}

	// Visit a parse tree produced by Parser#join_clause.
	VisitJoin_clause(ctx *Join_clauseContext) interface{}

	// Visit a parse tree produced by Parser#select_core.
	VisitSelect_core(ctx *Select_coreContext) interface{}

	// Visit a parse tree produced by Parser#factored_select_stmt.
	VisitFactored_select_stmt(ctx *Factored_select_stmtContext) interface{}

	// Visit a parse tree produced by Parser#simple_select_stmt.
	VisitSimple_select_stmt(ctx *Simple_select_stmtContext) interface{}

	// Visit a parse tree produced by Parser#compound_select_stmt.
	VisitCompound_select_stmt(ctx *Compound_select_stmtContext) interface{}

	// Visit a parse tree produced by Parser#table_or_subquery.
	VisitTable_or_subquery(ctx *Table_or_subqueryContext) interface{}

	// Visit a parse tree produced by Parser#result_column.
	VisitResult_column(ctx *Result_columnContext) interface{}

	// Visit a parse tree produced by Parser#join_operator.
	VisitJoin_operator(ctx *Join_operatorContext) interface{}

	// Visit a parse tree produced by Parser#join_constraint.
	VisitJoin_constraint(ctx *Join_constraintContext) interface{}

	// Visit a parse tree produced by Parser#compound_operator.
	VisitCompound_operator(ctx *Compound_operatorContext) interface{}

	// Visit a parse tree produced by Parser#update_stmt.
	VisitUpdate_stmt(ctx *Update_stmtContext) interface{}

	// Visit a parse tree produced by Parser#assignment_list.
	VisitAssignment_list(ctx *Assignment_listContext) interface{}

	// Visit a parse tree produced by Parser#assignment.
	VisitAssignment(ctx *AssignmentContext) interface{}

	// Visit a parse tree produced by Parser#column_name_list.
	VisitColumn_name_list(ctx *Column_name_listContext) interface{}

	// Visit a parse tree produced by Parser#update_stmt_limited.
	VisitUpdate_stmt_limited(ctx *Update_stmt_limitedContext) interface{}

	// Visit a parse tree produced by Parser#qualified_table_name.
	VisitQualified_table_name(ctx *Qualified_table_nameContext) interface{}

	// Visit a parse tree produced by Parser#vacuum_stmt.
	VisitVacuum_stmt(ctx *Vacuum_stmtContext) interface{}

	// Visit a parse tree produced by Parser#filter_clause.
	VisitFilter_clause(ctx *Filter_clauseContext) interface{}

	// Visit a parse tree produced by Parser#window_defn.
	VisitWindow_defn(ctx *Window_defnContext) interface{}

	// Visit a parse tree produced by Parser#over_clause.
	VisitOver_clause(ctx *Over_clauseContext) interface{}

	// Visit a parse tree produced by Parser#frame_spec.
	VisitFrame_spec(ctx *Frame_specContext) interface{}

	// Visit a parse tree produced by Parser#frame_clause.
	VisitFrame_clause(ctx *Frame_clauseContext) interface{}

	// Visit a parse tree produced by Parser#simple_function_invocation.
	VisitSimple_function_invocation(ctx *Simple_function_invocationContext) interface{}

	// Visit a parse tree produced by Parser#aggregate_function_invocation.
	VisitAggregate_function_invocation(ctx *Aggregate_function_invocationContext) interface{}

	// Visit a parse tree produced by Parser#window_function_invocation.
	VisitWindow_function_invocation(ctx *Window_function_invocationContext) interface{}

	// Visit a parse tree produced by Parser#common_table_stmt.
	VisitCommon_table_stmt(ctx *Common_table_stmtContext) interface{}

	// Visit a parse tree produced by Parser#order_by_stmt.
	VisitOrder_by_stmt(ctx *Order_by_stmtContext) interface{}

	// Visit a parse tree produced by Parser#limit_stmt.
	VisitLimit_stmt(ctx *Limit_stmtContext) interface{}

	// Visit a parse tree produced by Parser#ordering_term.
	VisitOrdering_term(ctx *Ordering_termContext) interface{}

	// Visit a parse tree produced by Parser#asc_desc.
	VisitAsc_desc(ctx *Asc_descContext) interface{}

	// Visit a parse tree produced by Parser#frame_left.
	VisitFrame_left(ctx *Frame_leftContext) interface{}

	// Visit a parse tree produced by Parser#frame_right.
	VisitFrame_right(ctx *Frame_rightContext) interface{}

	// Visit a parse tree produced by Parser#frame_single.
	VisitFrame_single(ctx *Frame_singleContext) interface{}

	// Visit a parse tree produced by Parser#window_function.
	VisitWindow_function(ctx *Window_functionContext) interface{}

	// Visit a parse tree produced by Parser#offset.
	VisitOffset(ctx *OffsetContext) interface{}

	// Visit a parse tree produced by Parser#default_value.
	VisitDefault_value(ctx *Default_valueContext) interface{}

	// Visit a parse tree produced by Parser#partition_by.
	VisitPartition_by(ctx *Partition_byContext) interface{}

	// Visit a parse tree produced by Parser#order_by_expr.
	VisitOrder_by_expr(ctx *Order_by_exprContext) interface{}

	// Visit a parse tree produced by Parser#order_by_expr_asc_desc.
	VisitOrder_by_expr_asc_desc(ctx *Order_by_expr_asc_descContext) interface{}

	// Visit a parse tree produced by Parser#expr_asc_desc.
	VisitExpr_asc_desc(ctx *Expr_asc_descContext) interface{}

	// Visit a parse tree produced by Parser#initial_select.
	VisitInitial_select(ctx *Initial_selectContext) interface{}

	// Visit a parse tree produced by Parser#recursive_select.
	VisitRecursive_select(ctx *Recursive_selectContext) interface{}

	// Visit a parse tree produced by Parser#unary_operator.
	VisitUnary_operator(ctx *Unary_operatorContext) interface{}

	// Visit a parse tree produced by Parser#error_message.
	VisitError_message(ctx *Error_messageContext) interface{}

	// Visit a parse tree produced by Parser#module_argument.
	VisitModule_argument(ctx *Module_argumentContext) interface{}

	// Visit a parse tree produced by Parser#column_alias.
	VisitColumn_alias(ctx *Column_aliasContext) interface{}

	// Visit a parse tree produced by Parser#keyword.
	VisitKeyword(ctx *KeywordContext) interface{}

	// Visit a parse tree produced by Parser#name.
	VisitName(ctx *NameContext) interface{}

	// Visit a parse tree produced by Parser#function_name.
	VisitFunction_name(ctx *Function_nameContext) interface{}

	// Visit a parse tree produced by Parser#schema_name.
	VisitSchema_name(ctx *Schema_nameContext) interface{}

	// Visit a parse tree produced by Parser#table_name.
	VisitTable_name(ctx *Table_nameContext) interface{}

	// Visit a parse tree produced by Parser#table_or_index_name.
	VisitTable_or_index_name(ctx *Table_or_index_nameContext) interface{}

	// Visit a parse tree produced by Parser#column_name.
	VisitColumn_name(ctx *Column_nameContext) interface{}

	// Visit a parse tree produced by Parser#collation_name.
	VisitCollation_name(ctx *Collation_nameContext) interface{}

	// Visit a parse tree produced by Parser#foreign_table.
	VisitForeign_table(ctx *Foreign_tableContext) interface{}

	// Visit a parse tree produced by Parser#index_name.
	VisitIndex_name(ctx *Index_nameContext) interface{}

	// Visit a parse tree produced by Parser#trigger_name.
	VisitTrigger_name(ctx *Trigger_nameContext) interface{}

	// Visit a parse tree produced by Parser#view_name.
	VisitView_name(ctx *View_nameContext) interface{}

	// Visit a parse tree produced by Parser#module_name.
	VisitModule_name(ctx *Module_nameContext) interface{}

	// Visit a parse tree produced by Parser#pragma_name.
	VisitPragma_name(ctx *Pragma_nameContext) interface{}

	// Visit a parse tree produced by Parser#savepoint_name.
	VisitSavepoint_name(ctx *Savepoint_nameContext) interface{}

	// Visit a parse tree produced by Parser#table_alias.
	VisitTable_alias(ctx *Table_aliasContext) interface{}

	// Visit a parse tree produced by Parser#transaction_name.
	VisitTransaction_name(ctx *Transaction_nameContext) interface{}

	// Visit a parse tree produced by Parser#window_name.
	VisitWindow_name(ctx *Window_nameContext) interface{}

	// Visit a parse tree produced by Parser#alias.
	VisitAlias(ctx *AliasContext) interface{}

	// Visit a parse tree produced by Parser#filename.
	VisitFilename(ctx *FilenameContext) interface{}

	// Visit a parse tree produced by Parser#base_window_name.
	VisitBase_window_name(ctx *Base_window_nameContext) interface{}

	// Visit a parse tree produced by Parser#simple_func.
	VisitSimple_func(ctx *Simple_funcContext) interface{}

	// Visit a parse tree produced by Parser#aggregate_func.
	VisitAggregate_func(ctx *Aggregate_funcContext) interface{}

	// Visit a parse tree produced by Parser#table_function_name.
	VisitTable_function_name(ctx *Table_function_nameContext) interface{}

	// Visit a parse tree produced by Parser#any_name.
	VisitAny_name(ctx *Any_nameContext) interface{}
}
