// Code generated from TSqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package tsqlparse // TSqlParser
import "github.com/antlr4-go/antlr/v4"

type BaseTSqlParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseTSqlParserVisitor) VisitTsql_file(ctx *Tsql_fileContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBatch(ctx *BatchContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBatch_level_statement(ctx *Batch_level_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSql_clauses(ctx *Sql_clausesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDml_clause(ctx *Dml_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDdl_clause(ctx *Ddl_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_statement(ctx *Backup_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCfl_statement(ctx *Cfl_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBlock_statement(ctx *Block_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBreak_statement(ctx *Break_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitContinue_statement(ctx *Continue_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGoto_statement(ctx *Goto_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReturn_statement(ctx *Return_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIf_statement(ctx *If_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitThrow_statement(ctx *Throw_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitThrow_error_number(ctx *Throw_error_numberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitThrow_message(ctx *Throw_messageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitThrow_state(ctx *Throw_stateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTry_catch_statement(ctx *Try_catch_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWaitfor_statement(ctx *Waitfor_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWhile_statement(ctx *While_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrint_statement(ctx *Print_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRaiseerror_statement(ctx *Raiseerror_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEmpty_statement(ctx *Empty_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAnother_statement(ctx *Another_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_application_role(ctx *Alter_application_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_xml_schema_collection(ctx *Alter_xml_schema_collectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_application_role(ctx *Create_application_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_aggregate(ctx *Drop_aggregateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_application_role(ctx *Drop_application_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly(ctx *Alter_assemblyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_start(ctx *Alter_assembly_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_clause(ctx *Alter_assembly_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_from_clause(ctx *Alter_assembly_from_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_from_clause_start(ctx *Alter_assembly_from_clause_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_drop_clause(ctx *Alter_assembly_drop_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_drop_multiple_files(ctx *Alter_assembly_drop_multiple_filesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_drop(ctx *Alter_assembly_dropContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_add_clause(ctx *Alter_assembly_add_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_asssembly_add_clause_start(ctx *Alter_asssembly_add_clause_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_client_file_clause(ctx *Alter_assembly_client_file_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_file_name(ctx *Alter_assembly_file_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_file_bits(ctx *Alter_assembly_file_bitsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_as(ctx *Alter_assembly_asContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_with_clause(ctx *Alter_assembly_with_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_assembly_with(ctx *Alter_assembly_withContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClient_assembly_specifier(ctx *Client_assembly_specifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAssembly_option(ctx *Assembly_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNetwork_file_share(ctx *Network_file_shareContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNetwork_computer(ctx *Network_computerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNetwork_file_start(ctx *Network_file_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFile_path(ctx *File_pathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFile_directory_path_separator(ctx *File_directory_path_separatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLocal_file(ctx *Local_fileContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLocal_drive(ctx *Local_driveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMultiple_local_files(ctx *Multiple_local_filesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMultiple_local_file_start(ctx *Multiple_local_file_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_assembly(ctx *Create_assemblyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_assembly(ctx *Drop_assemblyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_asymmetric_key(ctx *Alter_asymmetric_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_asymmetric_key_start(ctx *Alter_asymmetric_key_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAsymmetric_key_option(ctx *Asymmetric_key_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAsymmetric_key_option_start(ctx *Asymmetric_key_option_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAsymmetric_key_password_change_option(ctx *Asymmetric_key_password_change_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_asymmetric_key(ctx *Create_asymmetric_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_asymmetric_key(ctx *Drop_asymmetric_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_authorization(ctx *Alter_authorizationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAuthorization_grantee(ctx *Authorization_granteeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEntity_to(ctx *Entity_toContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColon_colon(ctx *Colon_colonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_authorization_start(ctx *Alter_authorization_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_authorization_for_sql_database(ctx *Alter_authorization_for_sql_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_authorization_for_azure_dw(ctx *Alter_authorization_for_azure_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_authorization_for_parallel_dw(ctx *Alter_authorization_for_parallel_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClass_type(ctx *Class_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClass_type_for_sql_database(ctx *Class_type_for_sql_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClass_type_for_azure_dw(ctx *Class_type_for_azure_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClass_type_for_parallel_dw(ctx *Class_type_for_parallel_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClass_type_for_grant(ctx *Class_type_for_grantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_availability_group(ctx *Drop_availability_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_availability_group(ctx *Alter_availability_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_availability_group_start(ctx *Alter_availability_group_startContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_availability_group_options(ctx *Alter_availability_group_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIp_v4_failover(ctx *Ip_v4_failoverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIp_v6_failover(ctx *Ip_v6_failoverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_broker_priority(ctx *Create_or_alter_broker_priorityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_broker_priority(ctx *Drop_broker_priorityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_certificate(ctx *Alter_certificateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_column_encryption_key(ctx *Alter_column_encryption_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_column_encryption_key(ctx *Create_column_encryption_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_certificate(ctx *Drop_certificateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_column_encryption_key(ctx *Drop_column_encryption_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_column_master_key(ctx *Drop_column_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_contract(ctx *Drop_contractContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_credential(ctx *Drop_credentialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_cryptograhic_provider(ctx *Drop_cryptograhic_providerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_database(ctx *Drop_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_database_audit_specification(ctx *Drop_database_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_database_encryption_key(ctx *Drop_database_encryption_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_database_scoped_credential(ctx *Drop_database_scoped_credentialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_default(ctx *Drop_defaultContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_endpoint(ctx *Drop_endpointContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_external_data_source(ctx *Drop_external_data_sourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_external_file_format(ctx *Drop_external_file_formatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_external_library(ctx *Drop_external_libraryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_external_resource_pool(ctx *Drop_external_resource_poolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_external_table(ctx *Drop_external_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_event_notifications(ctx *Drop_event_notificationsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_event_session(ctx *Drop_event_sessionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_fulltext_catalog(ctx *Drop_fulltext_catalogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_fulltext_index(ctx *Drop_fulltext_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_fulltext_stoplist(ctx *Drop_fulltext_stoplistContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_login(ctx *Drop_loginContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_master_key(ctx *Drop_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_message_type(ctx *Drop_message_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_partition_function(ctx *Drop_partition_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_partition_scheme(ctx *Drop_partition_schemeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_queue(ctx *Drop_queueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_remote_service_binding(ctx *Drop_remote_service_bindingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_resource_pool(ctx *Drop_resource_poolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_db_role(ctx *Drop_db_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_route(ctx *Drop_routeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_rule(ctx *Drop_ruleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_schema(ctx *Drop_schemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_search_property_list(ctx *Drop_search_property_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_security_policy(ctx *Drop_security_policyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_sequence(ctx *Drop_sequenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_server_audit(ctx *Drop_server_auditContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_server_audit_specification(ctx *Drop_server_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_server_role(ctx *Drop_server_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_service(ctx *Drop_serviceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_signature(ctx *Drop_signatureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_statistics_name_azure_dw_and_pdw(ctx *Drop_statistics_name_azure_dw_and_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_symmetric_key(ctx *Drop_symmetric_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_synonym(ctx *Drop_synonymContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_user(ctx *Drop_userContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_workload_group(ctx *Drop_workload_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_xml_schema_collection(ctx *Drop_xml_schema_collectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDisable_trigger(ctx *Disable_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEnable_trigger(ctx *Enable_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLock_table(ctx *Lock_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTruncate_table(ctx *Truncate_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_column_master_key(ctx *Create_column_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_credential(ctx *Alter_credentialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_credential(ctx *Create_credentialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_cryptographic_provider(ctx *Alter_cryptographic_providerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_cryptographic_provider(ctx *Create_cryptographic_providerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_endpoint(ctx *Create_endpointContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEndpoint_encryption_alogorithm_clause(ctx *Endpoint_encryption_alogorithm_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEndpoint_authentication_clause(ctx *Endpoint_authentication_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEndpoint_listener_clause(ctx *Endpoint_listener_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_event_notification(ctx *Create_event_notificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_event_session(ctx *Create_or_alter_event_sessionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEvent_session_predicate_expression(ctx *Event_session_predicate_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEvent_session_predicate_factor(ctx *Event_session_predicate_factorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEvent_session_predicate_leaf(ctx *Event_session_predicate_leafContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_external_data_source(ctx *Alter_external_data_sourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_external_library(ctx *Alter_external_libraryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_external_library(ctx *Create_external_libraryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_external_resource_pool(ctx *Alter_external_resource_poolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_external_resource_pool(ctx *Create_external_resource_poolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_fulltext_catalog(ctx *Alter_fulltext_catalogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_fulltext_catalog(ctx *Create_fulltext_catalogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_fulltext_stoplist(ctx *Alter_fulltext_stoplistContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_fulltext_stoplist(ctx *Create_fulltext_stoplistContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_login_sql_server(ctx *Alter_login_sql_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_login_sql_server(ctx *Create_login_sql_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_login_azure_sql(ctx *Alter_login_azure_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_login_azure_sql(ctx *Create_login_azure_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_login_azure_sql_dw_and_pdw(ctx *Alter_login_azure_sql_dw_and_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_login_pdw(ctx *Create_login_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_master_key_sql_server(ctx *Alter_master_key_sql_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_master_key_sql_server(ctx *Create_master_key_sql_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_master_key_azure_sql(ctx *Alter_master_key_azure_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_master_key_azure_sql(ctx *Create_master_key_azure_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_message_type(ctx *Alter_message_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_partition_function(ctx *Alter_partition_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_partition_scheme(ctx *Alter_partition_schemeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_remote_service_binding(ctx *Alter_remote_service_bindingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_remote_service_binding(ctx *Create_remote_service_bindingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_resource_pool(ctx *Create_resource_poolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_resource_governor(ctx *Alter_resource_governorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_database_audit_specification(ctx *Alter_database_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAudit_action_spec_group(ctx *Audit_action_spec_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAudit_action_specification(ctx *Audit_action_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAction_specification(ctx *Action_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAudit_class_name(ctx *Audit_class_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAudit_securable(ctx *Audit_securableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_db_role(ctx *Alter_db_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_database_audit_specification(ctx *Create_database_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_db_role(ctx *Create_db_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_route(ctx *Create_routeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_rule(ctx *Create_ruleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_schema_sql(ctx *Alter_schema_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_schema(ctx *Create_schemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_schema_azure_sql_dw_and_pdw(ctx *Create_schema_azure_sql_dw_and_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_schema_azure_sql_dw_and_pdw(ctx *Alter_schema_azure_sql_dw_and_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_search_property_list(ctx *Create_search_property_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_security_policy(ctx *Create_security_policyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_sequence(ctx *Alter_sequenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_sequence(ctx *Create_sequenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_server_audit(ctx *Alter_server_auditContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_server_audit(ctx *Create_server_auditContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_server_audit_specification(ctx *Alter_server_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_server_audit_specification(ctx *Create_server_audit_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_server_configuration(ctx *Alter_server_configurationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_server_role(ctx *Alter_server_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_server_role(ctx *Create_server_roleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_server_role_pdw(ctx *Alter_server_role_pdwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_service(ctx *Alter_serviceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpt_arg_clause(ctx *Opt_arg_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_service(ctx *Create_serviceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_service_master_key(ctx *Alter_service_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_symmetric_key(ctx *Alter_symmetric_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_synonym(ctx *Create_synonymContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_user(ctx *Alter_userContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_user(ctx *Create_userContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_user_azure_sql_dw(ctx *Create_user_azure_sql_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_user_azure_sql(ctx *Alter_user_azure_sqlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_workload_group(ctx *Alter_workload_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_workload_group(ctx *Create_workload_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_xml_schema_collection(ctx *Create_xml_schema_collectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_partition_function(ctx *Create_partition_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_partition_scheme(ctx *Create_partition_schemeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_queue(ctx *Create_queueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQueue_settings(ctx *Queue_settingsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_queue(ctx *Alter_queueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQueue_action(ctx *Queue_actionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQueue_rebuild_options(ctx *Queue_rebuild_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_contract(ctx *Create_contractContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitConversation_statement(ctx *Conversation_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMessage_statement(ctx *Message_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMerge_statement(ctx *Merge_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWhen_matches(ctx *When_matchesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMerge_matched(ctx *Merge_matchedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMerge_not_matched(ctx *Merge_not_matchedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDelete_statement(ctx *Delete_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDelete_statement_from(ctx *Delete_statement_fromContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitInsert_statement(ctx *Insert_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitInsert_statement_value(ctx *Insert_statement_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReceive_statement(ctx *Receive_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSelect_statement_standalone(ctx *Select_statement_standaloneContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSelect_statement(ctx *Select_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTime(ctx *TimeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_statement(ctx *Update_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOutput_clause(ctx *Output_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOutput_dml_list_elem(ctx *Output_dml_list_elemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_database(ctx *Create_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_index(ctx *Create_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_index_options(ctx *Create_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRelational_index_option(ctx *Relational_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_index(ctx *Alter_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitResumable_index_options(ctx *Resumable_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitResumable_index_option(ctx *Resumable_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReorganize_partition(ctx *Reorganize_partitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReorganize_options(ctx *Reorganize_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReorganize_option(ctx *Reorganize_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSet_index_options(ctx *Set_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSet_index_option(ctx *Set_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRebuild_partition(ctx *Rebuild_partitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRebuild_index_options(ctx *Rebuild_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRebuild_index_option(ctx *Rebuild_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSingle_partition_rebuild_index_options(ctx *Single_partition_rebuild_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSingle_partition_rebuild_index_option(ctx *Single_partition_rebuild_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOn_partitions(ctx *On_partitionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_columnstore_index(ctx *Create_columnstore_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_columnstore_index_options(ctx *Create_columnstore_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumnstore_index_option(ctx *Columnstore_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_nonclustered_columnstore_index(ctx *Create_nonclustered_columnstore_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_xml_index(ctx *Create_xml_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_index_options(ctx *Xml_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_index_option(ctx *Xml_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_procedure(ctx *Create_or_alter_procedureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAs_external_name(ctx *As_external_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_trigger(ctx *Create_or_alter_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_dml_trigger(ctx *Create_or_alter_dml_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDml_trigger_option(ctx *Dml_trigger_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDml_trigger_operation(ctx *Dml_trigger_operationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_ddl_trigger(ctx *Create_or_alter_ddl_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDdl_trigger_operation(ctx *Ddl_trigger_operationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_or_alter_function(ctx *Create_or_alter_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_body_returns_select(ctx *Func_body_returns_selectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_body_returns_table(ctx *Func_body_returns_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_body_returns_scalar(ctx *Func_body_returns_scalarContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitProcedure_param_default_value(ctx *Procedure_param_default_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitProcedure_param(ctx *Procedure_paramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitProcedure_option(ctx *Procedure_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunction_option(ctx *Function_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_statistics(ctx *Create_statisticsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_statistics(ctx *Update_statisticsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_statistics_options(ctx *Update_statistics_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_statistics_option(ctx *Update_statistics_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_table(ctx *Create_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_indices(ctx *Table_indicesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_options(ctx *Table_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_option(ctx *Table_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_table_index_options(ctx *Create_table_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_table_index_option(ctx *Create_table_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_view(ctx *Create_viewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitView_attribute(ctx *View_attributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_table(ctx *Alter_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSwitch_partition(ctx *Switch_partitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLow_priority_lock_wait(ctx *Low_priority_lock_waitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_database(ctx *Alter_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAdd_or_modify_files(ctx *Add_or_modify_filesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFilespec(ctx *FilespecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAdd_or_modify_filegroups(ctx *Add_or_modify_filegroupsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFilegroup_updatability_option(ctx *Filegroup_updatability_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDatabase_optionspec(ctx *Database_optionspecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAuto_option(ctx *Auto_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitChange_tracking_option(ctx *Change_tracking_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitChange_tracking_option_list(ctx *Change_tracking_option_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitContainment_option(ctx *Containment_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCursor_option(ctx *Cursor_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_endpoint(ctx *Alter_endpointContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDatabase_mirroring_option(ctx *Database_mirroring_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMirroring_set_option(ctx *Mirroring_set_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMirroring_partner(ctx *Mirroring_partnerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMirroring_witness(ctx *Mirroring_witnessContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWitness_partner_equal(ctx *Witness_partner_equalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPartner_option(ctx *Partner_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWitness_option(ctx *Witness_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWitness_server(ctx *Witness_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPartner_server(ctx *Partner_serverContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMirroring_host_port_seperator(ctx *Mirroring_host_port_seperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPartner_server_tcp_prefix(ctx *Partner_server_tcp_prefixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPort_number(ctx *Port_numberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHost(ctx *HostContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDate_correlation_optimization_option(ctx *Date_correlation_optimization_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDb_encryption_option(ctx *Db_encryption_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDb_state_option(ctx *Db_state_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDb_update_option(ctx *Db_update_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDb_user_access_option(ctx *Db_user_access_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDelayed_durability_option(ctx *Delayed_durability_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExternal_access_option(ctx *External_access_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHadr_options(ctx *Hadr_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMixed_page_allocation_option(ctx *Mixed_page_allocation_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitParameterization_option(ctx *Parameterization_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRecovery_option(ctx *Recovery_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitService_broker_option(ctx *Service_broker_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSnapshot_option(ctx *Snapshot_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSql_option(ctx *Sql_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTarget_recovery_time_option(ctx *Target_recovery_time_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTermination(ctx *TerminationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_index(ctx *Drop_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_relational_or_xml_or_spatial_index(ctx *Drop_relational_or_xml_or_spatial_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_backward_compatible_index(ctx *Drop_backward_compatible_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_procedure(ctx *Drop_procedureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_trigger(ctx *Drop_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_dml_trigger(ctx *Drop_dml_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_ddl_trigger(ctx *Drop_ddl_triggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_function(ctx *Drop_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_statistics(ctx *Drop_statisticsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_table(ctx *Drop_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_view(ctx *Drop_viewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_type(ctx *Create_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDrop_type(ctx *Drop_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRowset_function_limited(ctx *Rowset_function_limitedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpenquery(ctx *OpenqueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpendatasource(ctx *OpendatasourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeclare_statement(ctx *Declare_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_declaration(ctx *Xml_declarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCursor_statement(ctx *Cursor_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_database(ctx *Backup_databaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_log(ctx *Backup_logContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_certificate(ctx *Backup_certificateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_master_key(ctx *Backup_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBackup_service_master_key(ctx *Backup_service_master_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKill_statement(ctx *Kill_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKill_process(ctx *Kill_processContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKill_query_notification(ctx *Kill_query_notificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKill_stats_job(ctx *Kill_stats_jobContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_statement(ctx *Execute_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_body_batch(ctx *Execute_body_batchContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_body(ctx *Execute_bodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_statement_arg(ctx *Execute_statement_argContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_statement_arg_named(ctx *Execute_statement_arg_namedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_statement_arg_unnamed(ctx *Execute_statement_arg_unnamedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_parameter(ctx *Execute_parameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_var_string(ctx *Execute_var_stringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSecurity_statement(ctx *Security_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrincipal_id(ctx *Principal_idContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_certificate(ctx *Create_certificateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExisting_keys(ctx *Existing_keysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrivate_key_options(ctx *Private_key_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGenerate_new_keys(ctx *Generate_new_keysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDate_options(ctx *Date_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpen_key(ctx *Open_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClose_key(ctx *Close_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_key(ctx *Create_keyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKey_options(ctx *Key_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlgorithm(ctx *AlgorithmContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEncryption_mechanism(ctx *Encryption_mechanismContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDecryption_mechanism(ctx *Decryption_mechanismContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGrant_permission(ctx *Grant_permissionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSet_statement(ctx *Set_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTransaction_statement(ctx *Transaction_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGo_statement(ctx *Go_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUse_statement(ctx *Use_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSetuser_statement(ctx *Setuser_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitReconfigure_statement(ctx *Reconfigure_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitShutdown_statement(ctx *Shutdown_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCheckpoint_statement(ctx *Checkpoint_statementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkalloc_option(ctx *Dbcc_checkalloc_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkalloc(ctx *Dbcc_checkallocContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkcatalog(ctx *Dbcc_checkcatalogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkconstraints_option(ctx *Dbcc_checkconstraints_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkconstraints(ctx *Dbcc_checkconstraintsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkdb_table_option(ctx *Dbcc_checkdb_table_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkdb(ctx *Dbcc_checkdbContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkfilegroup_option(ctx *Dbcc_checkfilegroup_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checkfilegroup(ctx *Dbcc_checkfilegroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_checktable(ctx *Dbcc_checktableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_cleantable(ctx *Dbcc_cleantableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_clonedatabase_option(ctx *Dbcc_clonedatabase_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_clonedatabase(ctx *Dbcc_clonedatabaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_pdw_showspaceused(ctx *Dbcc_pdw_showspaceusedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_proccache(ctx *Dbcc_proccacheContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_showcontig_option(ctx *Dbcc_showcontig_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_showcontig(ctx *Dbcc_showcontigContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_shrinklog(ctx *Dbcc_shrinklogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_dbreindex(ctx *Dbcc_dbreindexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_dll_free(ctx *Dbcc_dll_freeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_dropcleanbuffers(ctx *Dbcc_dropcleanbuffersContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDbcc_clause(ctx *Dbcc_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExecute_clause(ctx *Execute_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeclare_local(ctx *Declare_localContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_type_definition(ctx *Table_type_definitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_type_indices(ctx *Table_type_indicesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_type_definition(ctx *Xml_type_definitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_schema_collection(ctx *Xml_schema_collectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_def_table_constraints(ctx *Column_def_table_constraintsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_def_table_constraint(ctx *Column_def_table_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_definition(ctx *Column_definitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_definition_element(ctx *Column_definition_elementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_modifier(ctx *Column_modifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMaterialized_column_definition(ctx *Materialized_column_definitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_constraint(ctx *Column_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_index(ctx *Column_indexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOn_partition_or_filegroup(ctx *On_partition_or_filegroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_constraint(ctx *Table_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitConnection_node(ctx *Connection_nodeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrimary_key_options(ctx *Primary_key_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitForeign_key_options(ctx *Foreign_key_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCheck_constraint(ctx *Check_constraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOn_delete(ctx *On_deleteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOn_update(ctx *On_updateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_table_index_options(ctx *Alter_table_index_optionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAlter_table_index_option(ctx *Alter_table_index_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeclare_cursor(ctx *Declare_cursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeclare_set_cursor_common(ctx *Declare_set_cursor_commonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeclare_set_cursor_common_partial(ctx *Declare_set_cursor_common_partialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFetch_cursor(ctx *Fetch_cursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSet_special(ctx *Set_specialContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSpecial_list(ctx *Special_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitConstant_LOCAL_ID(ctx *Constant_LOCAL_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitParameter(ctx *ParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTime_zone(ctx *Time_zoneContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrimitive_expression(ctx *Primitive_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCase_expression(ctx *Case_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUnary_operator_expression(ctx *Unary_operator_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBracket_expression(ctx *Bracket_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSubquery(ctx *SubqueryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWith_expression(ctx *With_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCommon_table_expression(ctx *Common_table_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_elem(ctx *Update_elemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUpdate_elem_merge(ctx *Update_elem_mergeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSearch_condition(ctx *Search_conditionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPredicate(ctx *PredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQuery_expression(ctx *Query_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSql_union(ctx *Sql_unionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQuery_specification(ctx *Query_specificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTop_clause(ctx *Top_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTop_percent(ctx *Top_percentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTop_count(ctx *Top_countContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOrder_by_clause(ctx *Order_by_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSelect_order_by_clause(ctx *Select_order_by_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFor_clause(ctx *For_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_common_directives(ctx *Xml_common_directivesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOrder_by_expression(ctx *Order_by_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGrouping_sets_item(ctx *Grouping_sets_itemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGroup_by_item(ctx *Group_by_itemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOption_clause(ctx *Option_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOption(ctx *OptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOptimize_for_arg(ctx *Optimize_for_argContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSelect_list(ctx *Select_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUdt_method_arguments(ctx *Udt_method_argumentsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAsterisk(ctx *AsteriskContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUdt_elem(ctx *Udt_elemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExpression_elem(ctx *Expression_elemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSelect_list_elem(ctx *Select_list_elemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_sources(ctx *Table_sourcesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNon_ansi_join(ctx *Non_ansi_joinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_source(ctx *Table_sourceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_source_item(ctx *Table_source_itemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpen_xml(ctx *Open_xmlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOpen_json(ctx *Open_jsonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJson_declaration(ctx *Json_declarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJson_column_declaration(ctx *Json_column_declarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSchema_declaration(ctx *Schema_declarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_declaration(ctx *Column_declarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitChange_table(ctx *Change_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitChange_table_changes(ctx *Change_table_changesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitChange_table_version(ctx *Change_table_versionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJoin_part(ctx *Join_partContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJoin_on(ctx *Join_onContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCross_join(ctx *Cross_joinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitApply_(ctx *Apply_Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPivot(ctx *PivotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUnpivot(ctx *UnpivotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPivot_clause(ctx *Pivot_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUnpivot_clause(ctx *Unpivot_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFull_column_name_list(ctx *Full_column_name_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRowset_function(ctx *Rowset_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBulk_option(ctx *Bulk_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDerived_table(ctx *Derived_tableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRANKING_WINDOWED_FUNC(ctx *RANKING_WINDOWED_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAGGREGATE_WINDOWED_FUNC(ctx *AGGREGATE_WINDOWED_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitANALYTIC_WINDOWED_FUNC(ctx *ANALYTIC_WINDOWED_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBUILT_IN_FUNC(ctx *BUILT_IN_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSCALAR_FUNCTION(ctx *SCALAR_FUNCTIONContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFREE_TEXT(ctx *FREE_TEXTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPARTITION_FUNC(ctx *PARTITION_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHIERARCHYID_METHOD(ctx *HIERARCHYID_METHODContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPartition_function(ctx *Partition_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFreetext_function(ctx *Freetext_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFreetext_predicate(ctx *Freetext_predicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJson_key_value(ctx *Json_key_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJson_null_clause(ctx *Json_null_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAPP_NAME(ctx *APP_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAPPLOCK_MODE(ctx *APPLOCK_MODEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAPPLOCK_TEST(ctx *APPLOCK_TESTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitASSEMBLYPROPERTY(ctx *ASSEMBLYPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOL_LENGTH(ctx *COL_LENGTHContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOL_NAME(ctx *COL_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOLUMNPROPERTY(ctx *COLUMNPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATABASEPROPERTYEX(ctx *DATABASEPROPERTYEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDB_ID(ctx *DB_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDB_NAME(ctx *DB_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILE_ID(ctx *FILE_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILE_IDEX(ctx *FILE_IDEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILE_NAME(ctx *FILE_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILEGROUP_ID(ctx *FILEGROUP_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILEGROUP_NAME(ctx *FILEGROUP_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILEGROUPPROPERTY(ctx *FILEGROUPPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILEPROPERTY(ctx *FILEPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFILEPROPERTYEX(ctx *FILEPROPERTYEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFULLTEXTCATALOGPROPERTY(ctx *FULLTEXTCATALOGPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFULLTEXTSERVICEPROPERTY(ctx *FULLTEXTSERVICEPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitINDEX_COL(ctx *INDEX_COLContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitINDEXKEY_PROPERTY(ctx *INDEXKEY_PROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitINDEXPROPERTY(ctx *INDEXPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNEXT_VALUE_FOR(ctx *NEXT_VALUE_FORContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECT_DEFINITION(ctx *OBJECT_DEFINITIONContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECT_ID(ctx *OBJECT_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECT_NAME(ctx *OBJECT_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECT_SCHEMA_NAME(ctx *OBJECT_SCHEMA_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECTPROPERTY(ctx *OBJECTPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOBJECTPROPERTYEX(ctx *OBJECTPROPERTYEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitORIGINAL_DB_NAME(ctx *ORIGINAL_DB_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPARSENAME(ctx *PARSENAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSCHEMA_ID(ctx *SCHEMA_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSCHEMA_NAME(ctx *SCHEMA_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSCOPE_IDENTITY(ctx *SCOPE_IDENTITYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSERVERPROPERTY(ctx *SERVERPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSTATS_DATE(ctx *STATS_DATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTYPE_ID(ctx *TYPE_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTYPE_NAME(ctx *TYPE_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTYPEPROPERTY(ctx *TYPEPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitASCII(ctx *ASCIIContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCHAR(ctx *CHARContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCHARINDEX(ctx *CHARINDEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCONCAT(ctx *CONCATContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCONCAT_WS(ctx *CONCAT_WSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDIFFERENCE(ctx *DIFFERENCEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFORMAT(ctx *FORMATContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLEFT(ctx *LEFTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLEN(ctx *LENContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLOWER(ctx *LOWERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLTRIM(ctx *LTRIMContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNCHAR(ctx *NCHARContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPATINDEX(ctx *PATINDEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQUOTENAME(ctx *QUOTENAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitREPLACE(ctx *REPLACEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitREPLICATE(ctx *REPLICATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitREVERSE(ctx *REVERSEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRIGHT(ctx *RIGHTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRTRIM(ctx *RTRIMContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSOUNDEX(ctx *SOUNDEXContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSPACE(ctx *SPACEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSTR(ctx *STRContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSTRINGAGG(ctx *STRINGAGGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSTRING_ESCAPE(ctx *STRING_ESCAPEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSTUFF(ctx *STUFFContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSUBSTRING(ctx *SUBSTRINGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTRANSLATE(ctx *TRANSLATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTRIM(ctx *TRIMContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUNICODE(ctx *UNICODEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUPPER(ctx *UPPERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBINARY_CHECKSUM(ctx *BINARY_CHECKSUMContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCHECKSUM(ctx *CHECKSUMContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOMPRESS(ctx *COMPRESSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCONNECTIONPROPERTY(ctx *CONNECTIONPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCONTEXT_INFO(ctx *CONTEXT_INFOContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_REQUEST_ID(ctx *CURRENT_REQUEST_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_TRANSACTION_ID(ctx *CURRENT_TRANSACTION_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDECOMPRESS(ctx *DECOMPRESSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_LINE(ctx *ERROR_LINEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_MESSAGE(ctx *ERROR_MESSAGEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_NUMBER(ctx *ERROR_NUMBERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_PROCEDURE(ctx *ERROR_PROCEDUREContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_SEVERITY(ctx *ERROR_SEVERITYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitERROR_STATE(ctx *ERROR_STATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFORMATMESSAGE(ctx *FORMATMESSAGEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGET_FILESTREAM_TRANSACTION_CONTEXT(ctx *GET_FILESTREAM_TRANSACTION_CONTEXTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGETANSINULL(ctx *GETANSINULLContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHOST_ID(ctx *HOST_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHOST_NAME(ctx *HOST_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitISNULL(ctx *ISNULLContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitISNUMERIC(ctx *ISNUMERICContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMIN_ACTIVE_ROWVERSION(ctx *MIN_ACTIVE_ROWVERSIONContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNEWID(ctx *NEWIDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNEWSEQUENTIALID(ctx *NEWSEQUENTIALIDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitROWCOUNT_BIG(ctx *ROWCOUNT_BIGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSESSION_CONTEXT(ctx *SESSION_CONTEXTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXACT_STATE(ctx *XACT_STATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCAST(ctx *CASTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTRY_CAST(ctx *TRY_CASTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCONVERT(ctx *CONVERTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOALESCE(ctx *COALESCEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURSOR_ROWS(ctx *CURSOR_ROWSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFETCH_STATUS(ctx *FETCH_STATUSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURSOR_STATUS(ctx *CURSOR_STATUSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCERT_ID(ctx *CERT_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATALENGTH(ctx *DATALENGTHContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIDENT_CURRENT(ctx *IDENT_CURRENTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIDENT_INCR(ctx *IDENT_INCRContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIDENT_SEED(ctx *IDENT_SEEDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIDENTITY(ctx *IDENTITYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSQL_VARIANT_PROPERTY(ctx *SQL_VARIANT_PROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_DATE(ctx *CURRENT_DATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_TIMESTAMP(ctx *CURRENT_TIMESTAMPContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_TIMEZONE(ctx *CURRENT_TIMEZONEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_TIMEZONE_ID(ctx *CURRENT_TIMEZONE_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATE_BUCKET(ctx *DATE_BUCKETContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATEADD(ctx *DATEADDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATEDIFF(ctx *DATEDIFFContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATEDIFF_BIG(ctx *DATEDIFF_BIGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATEFROMPARTS(ctx *DATEFROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATENAME(ctx *DATENAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATEPART(ctx *DATEPARTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATETIME2FROMPARTS(ctx *DATETIME2FROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATETIMEFROMPARTS(ctx *DATETIMEFROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATETIMEOFFSETFROMPARTS(ctx *DATETIMEOFFSETFROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATETRUNC(ctx *DATETRUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDAY(ctx *DAYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEOMONTH(ctx *EOMONTHContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGETDATE(ctx *GETDATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGETUTCDATE(ctx *GETUTCDATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitISDATE(ctx *ISDATEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMONTH(ctx *MONTHContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSMALLDATETIMEFROMPARTS(ctx *SMALLDATETIMEFROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSWITCHOFFSET(ctx *SWITCHOFFSETContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSYSDATETIME(ctx *SYSDATETIMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSYSDATETIMEOFFSET(ctx *SYSDATETIMEOFFSETContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSYSUTCDATETIME(ctx *SYSUTCDATETIMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTIMEFROMPARTS(ctx *TIMEFROMPARTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTODATETIMEOFFSET(ctx *TODATETIMEOFFSETContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitYEAR(ctx *YEARContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNULLIF(ctx *NULLIFContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPARSE(ctx *PARSEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXML_DATA_TYPE_FUNC(ctx *XML_DATA_TYPE_FUNCContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIIF(ctx *IIFContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitISJSON(ctx *ISJSONContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_OBJECT(ctx *JSON_OBJECTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_ARRAY(ctx *JSON_ARRAYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_VALUE(ctx *JSON_VALUEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_QUERY(ctx *JSON_QUERYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_MODIFY(ctx *JSON_MODIFYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitJSON_PATH_EXISTS(ctx *JSON_PATH_EXISTSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitABS(ctx *ABSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitACOS(ctx *ACOSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitASIN(ctx *ASINContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitATAN(ctx *ATANContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitATN2(ctx *ATN2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCEILING(ctx *CEILINGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOS(ctx *COSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCOT(ctx *COTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDEGREES(ctx *DEGREESContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEXP(ctx *EXPContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFLOOR(ctx *FLOORContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLOG(ctx *LOGContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLOG10(ctx *LOG10Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPI(ctx *PIContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPOWER(ctx *POWERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRADIANS(ctx *RADIANSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRAND(ctx *RANDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitROUND(ctx *ROUNDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitMATH_SIGN(ctx *MATH_SIGNContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSIN(ctx *SINContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSQRT(ctx *SQRTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSQUARE(ctx *SQUAREContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTAN(ctx *TANContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGREATEST(ctx *GREATESTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLEAST(ctx *LEASTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCERTENCODED(ctx *CERTENCODEDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCERTPRIVATEKEY(ctx *CERTPRIVATEKEYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCURRENT_USER(ctx *CURRENT_USERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDATABASE_PRINCIPAL_ID(ctx *DATABASE_PRINCIPAL_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHAS_DBACCESS(ctx *HAS_DBACCESSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHAS_PERMS_BY_NAME(ctx *HAS_PERMS_BY_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIS_MEMBER(ctx *IS_MEMBERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIS_ROLEMEMBER(ctx *IS_ROLEMEMBERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIS_SRVROLEMEMBER(ctx *IS_SRVROLEMEMBERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitLOGINPROPERTY(ctx *LOGINPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitORIGINAL_LOGIN(ctx *ORIGINAL_LOGINContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPERMISSIONS(ctx *PERMISSIONSContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPWDENCRYPT(ctx *PWDENCRYPTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPWDCOMPARE(ctx *PWDCOMPAREContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSESSION_USER(ctx *SESSION_USERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSESSIONPROPERTY(ctx *SESSIONPROPERTYContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSUSER_ID(ctx *SUSER_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSUSER_SNAME(ctx *SUSER_SNAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSUSER_SID(ctx *SUSER_SIDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSYSTEM_USER(ctx *SYSTEM_USERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUSER(ctx *USERContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUSER_ID(ctx *USER_IDContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitUSER_NAME(ctx *USER_NAMEContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitXml_data_type_methods(ctx *Xml_data_type_methodsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDateparts_9(ctx *Dateparts_9Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDateparts_12(ctx *Dateparts_12Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDateparts_15(ctx *Dateparts_15Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDateparts_datetrunc(ctx *Dateparts_datetruncContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitValue_method(ctx *Value_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitValue_call(ctx *Value_callContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQuery_method(ctx *Query_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQuery_call(ctx *Query_callContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExist_method(ctx *Exist_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExist_call(ctx *Exist_callContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitModify_method(ctx *Modify_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitModify_call(ctx *Modify_callContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHierarchyid_call(ctx *Hierarchyid_callContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitHierarchyid_static_method(ctx *Hierarchyid_static_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNodes_method(ctx *Nodes_methodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSwitch_section(ctx *Switch_sectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSwitch_search_condition_section(ctx *Switch_search_condition_sectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAs_column_alias(ctx *As_column_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAs_table_alias(ctx *As_table_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_alias(ctx *Table_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWith_table_hints(ctx *With_table_hintsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDeprecated_table_hint(ctx *Deprecated_table_hintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSybase_legacy_hints(ctx *Sybase_legacy_hintsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSybase_legacy_hint(ctx *Sybase_legacy_hintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_hint(ctx *Table_hintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitIndex_value(ctx *Index_valueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_alias_list(ctx *Column_alias_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_alias(ctx *Column_aliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_value_constructor(ctx *Table_value_constructorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitExpression_list_(ctx *Expression_list_Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRanking_windowed_function(ctx *Ranking_windowed_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAggregate_windowed_function(ctx *Aggregate_windowed_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAnalytic_windowed_function(ctx *Analytic_windowed_functionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAll_distinct_expression(ctx *All_distinct_expressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOver_clause(ctx *Over_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitRow_or_range_clause(ctx *Row_or_range_clauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWindow_frame_extent(ctx *Window_frame_extentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWindow_frame_bound(ctx *Window_frame_boundContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWindow_frame_preceding(ctx *Window_frame_precedingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWindow_frame_following(ctx *Window_frame_followingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCreate_database_option(ctx *Create_database_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDatabase_filestream_option(ctx *Database_filestream_optionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDatabase_file_spec(ctx *Database_file_specContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFile_group(ctx *File_groupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFile_spec(ctx *File_specContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEntity_name(ctx *Entity_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEntity_name_for_azure_dw(ctx *Entity_name_for_azure_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEntity_name_for_parallel_dw(ctx *Entity_name_for_parallel_dwContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFull_table_name(ctx *Full_table_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitTable_name(ctx *Table_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSimple_name(ctx *Simple_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_proc_name_schema(ctx *Func_proc_name_schemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_proc_name_database_schema(ctx *Func_proc_name_database_schemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFunc_proc_name_server_database_schema(ctx *Func_proc_name_server_database_schemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitDdl_object(ctx *Ddl_objectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFull_column_name(ctx *Full_column_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_name_list_with_order(ctx *Column_name_list_with_orderContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitInsert_column_name_list(ctx *Insert_column_name_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitInsert_column_id(ctx *Insert_column_idContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitColumn_name_list(ctx *Column_name_listContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitCursor_name(ctx *Cursor_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitOn_off(ctx *On_offContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitClustered(ctx *ClusteredContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitNull_notnull(ctx *Null_notnullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitScalar_function_name(ctx *Scalar_function_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBegin_conversation_timer(ctx *Begin_conversation_timerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitBegin_conversation_dialog(ctx *Begin_conversation_dialogContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitContract_name(ctx *Contract_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitService_name(ctx *Service_nameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitEnd_conversation(ctx *End_conversationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitWaitfor_conversation(ctx *Waitfor_conversationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitGet_conversation(ctx *Get_conversationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitQueue_id(ctx *Queue_idContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSend_conversation(ctx *Send_conversationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitData_type(ctx *Data_typeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitConstant(ctx *ConstantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitPrimitive_constant(ctx *Primitive_constantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitKeyword(ctx *KeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitId_(ctx *Id_Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitSimple_id(ctx *Simple_idContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitId_or_string(ctx *Id_or_stringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitComparison_operator(ctx *Comparison_operatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitAssignment_operator(ctx *Assignment_operatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseTSqlParserVisitor) VisitFile_size(ctx *File_sizeContext) interface{} {
	return v.VisitChildren(ctx)
}
