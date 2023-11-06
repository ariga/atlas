// Code generated from TSqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package tsqlparse // TSqlParser
import "github.com/antlr4-go/antlr/v4"

// TSqlParserListener is a complete listener for a parse tree produced by TSqlParser.
type TSqlParserListener interface {
	antlr.ParseTreeListener

	// EnterTsql_file is called when entering the tsql_file production.
	EnterTsql_file(c *Tsql_fileContext)

	// EnterBatch is called when entering the batch production.
	EnterBatch(c *BatchContext)

	// EnterBatch_level_statement is called when entering the batch_level_statement production.
	EnterBatch_level_statement(c *Batch_level_statementContext)

	// EnterSql_clauses is called when entering the sql_clauses production.
	EnterSql_clauses(c *Sql_clausesContext)

	// EnterDml_clause is called when entering the dml_clause production.
	EnterDml_clause(c *Dml_clauseContext)

	// EnterDdl_clause is called when entering the ddl_clause production.
	EnterDdl_clause(c *Ddl_clauseContext)

	// EnterBackup_statement is called when entering the backup_statement production.
	EnterBackup_statement(c *Backup_statementContext)

	// EnterCfl_statement is called when entering the cfl_statement production.
	EnterCfl_statement(c *Cfl_statementContext)

	// EnterBlock_statement is called when entering the block_statement production.
	EnterBlock_statement(c *Block_statementContext)

	// EnterBreak_statement is called when entering the break_statement production.
	EnterBreak_statement(c *Break_statementContext)

	// EnterContinue_statement is called when entering the continue_statement production.
	EnterContinue_statement(c *Continue_statementContext)

	// EnterGoto_statement is called when entering the goto_statement production.
	EnterGoto_statement(c *Goto_statementContext)

	// EnterReturn_statement is called when entering the return_statement production.
	EnterReturn_statement(c *Return_statementContext)

	// EnterIf_statement is called when entering the if_statement production.
	EnterIf_statement(c *If_statementContext)

	// EnterThrow_statement is called when entering the throw_statement production.
	EnterThrow_statement(c *Throw_statementContext)

	// EnterThrow_error_number is called when entering the throw_error_number production.
	EnterThrow_error_number(c *Throw_error_numberContext)

	// EnterThrow_message is called when entering the throw_message production.
	EnterThrow_message(c *Throw_messageContext)

	// EnterThrow_state is called when entering the throw_state production.
	EnterThrow_state(c *Throw_stateContext)

	// EnterTry_catch_statement is called when entering the try_catch_statement production.
	EnterTry_catch_statement(c *Try_catch_statementContext)

	// EnterWaitfor_statement is called when entering the waitfor_statement production.
	EnterWaitfor_statement(c *Waitfor_statementContext)

	// EnterWhile_statement is called when entering the while_statement production.
	EnterWhile_statement(c *While_statementContext)

	// EnterPrint_statement is called when entering the print_statement production.
	EnterPrint_statement(c *Print_statementContext)

	// EnterRaiseerror_statement is called when entering the raiseerror_statement production.
	EnterRaiseerror_statement(c *Raiseerror_statementContext)

	// EnterEmpty_statement is called when entering the empty_statement production.
	EnterEmpty_statement(c *Empty_statementContext)

	// EnterAnother_statement is called when entering the another_statement production.
	EnterAnother_statement(c *Another_statementContext)

	// EnterAlter_application_role is called when entering the alter_application_role production.
	EnterAlter_application_role(c *Alter_application_roleContext)

	// EnterAlter_xml_schema_collection is called when entering the alter_xml_schema_collection production.
	EnterAlter_xml_schema_collection(c *Alter_xml_schema_collectionContext)

	// EnterCreate_application_role is called when entering the create_application_role production.
	EnterCreate_application_role(c *Create_application_roleContext)

	// EnterDrop_aggregate is called when entering the drop_aggregate production.
	EnterDrop_aggregate(c *Drop_aggregateContext)

	// EnterDrop_application_role is called when entering the drop_application_role production.
	EnterDrop_application_role(c *Drop_application_roleContext)

	// EnterAlter_assembly is called when entering the alter_assembly production.
	EnterAlter_assembly(c *Alter_assemblyContext)

	// EnterAlter_assembly_start is called when entering the alter_assembly_start production.
	EnterAlter_assembly_start(c *Alter_assembly_startContext)

	// EnterAlter_assembly_clause is called when entering the alter_assembly_clause production.
	EnterAlter_assembly_clause(c *Alter_assembly_clauseContext)

	// EnterAlter_assembly_from_clause is called when entering the alter_assembly_from_clause production.
	EnterAlter_assembly_from_clause(c *Alter_assembly_from_clauseContext)

	// EnterAlter_assembly_from_clause_start is called when entering the alter_assembly_from_clause_start production.
	EnterAlter_assembly_from_clause_start(c *Alter_assembly_from_clause_startContext)

	// EnterAlter_assembly_drop_clause is called when entering the alter_assembly_drop_clause production.
	EnterAlter_assembly_drop_clause(c *Alter_assembly_drop_clauseContext)

	// EnterAlter_assembly_drop_multiple_files is called when entering the alter_assembly_drop_multiple_files production.
	EnterAlter_assembly_drop_multiple_files(c *Alter_assembly_drop_multiple_filesContext)

	// EnterAlter_assembly_drop is called when entering the alter_assembly_drop production.
	EnterAlter_assembly_drop(c *Alter_assembly_dropContext)

	// EnterAlter_assembly_add_clause is called when entering the alter_assembly_add_clause production.
	EnterAlter_assembly_add_clause(c *Alter_assembly_add_clauseContext)

	// EnterAlter_asssembly_add_clause_start is called when entering the alter_asssembly_add_clause_start production.
	EnterAlter_asssembly_add_clause_start(c *Alter_asssembly_add_clause_startContext)

	// EnterAlter_assembly_client_file_clause is called when entering the alter_assembly_client_file_clause production.
	EnterAlter_assembly_client_file_clause(c *Alter_assembly_client_file_clauseContext)

	// EnterAlter_assembly_file_name is called when entering the alter_assembly_file_name production.
	EnterAlter_assembly_file_name(c *Alter_assembly_file_nameContext)

	// EnterAlter_assembly_file_bits is called when entering the alter_assembly_file_bits production.
	EnterAlter_assembly_file_bits(c *Alter_assembly_file_bitsContext)

	// EnterAlter_assembly_as is called when entering the alter_assembly_as production.
	EnterAlter_assembly_as(c *Alter_assembly_asContext)

	// EnterAlter_assembly_with_clause is called when entering the alter_assembly_with_clause production.
	EnterAlter_assembly_with_clause(c *Alter_assembly_with_clauseContext)

	// EnterAlter_assembly_with is called when entering the alter_assembly_with production.
	EnterAlter_assembly_with(c *Alter_assembly_withContext)

	// EnterClient_assembly_specifier is called when entering the client_assembly_specifier production.
	EnterClient_assembly_specifier(c *Client_assembly_specifierContext)

	// EnterAssembly_option is called when entering the assembly_option production.
	EnterAssembly_option(c *Assembly_optionContext)

	// EnterNetwork_file_share is called when entering the network_file_share production.
	EnterNetwork_file_share(c *Network_file_shareContext)

	// EnterNetwork_computer is called when entering the network_computer production.
	EnterNetwork_computer(c *Network_computerContext)

	// EnterNetwork_file_start is called when entering the network_file_start production.
	EnterNetwork_file_start(c *Network_file_startContext)

	// EnterFile_path is called when entering the file_path production.
	EnterFile_path(c *File_pathContext)

	// EnterFile_directory_path_separator is called when entering the file_directory_path_separator production.
	EnterFile_directory_path_separator(c *File_directory_path_separatorContext)

	// EnterLocal_file is called when entering the local_file production.
	EnterLocal_file(c *Local_fileContext)

	// EnterLocal_drive is called when entering the local_drive production.
	EnterLocal_drive(c *Local_driveContext)

	// EnterMultiple_local_files is called when entering the multiple_local_files production.
	EnterMultiple_local_files(c *Multiple_local_filesContext)

	// EnterMultiple_local_file_start is called when entering the multiple_local_file_start production.
	EnterMultiple_local_file_start(c *Multiple_local_file_startContext)

	// EnterCreate_assembly is called when entering the create_assembly production.
	EnterCreate_assembly(c *Create_assemblyContext)

	// EnterDrop_assembly is called when entering the drop_assembly production.
	EnterDrop_assembly(c *Drop_assemblyContext)

	// EnterAlter_asymmetric_key is called when entering the alter_asymmetric_key production.
	EnterAlter_asymmetric_key(c *Alter_asymmetric_keyContext)

	// EnterAlter_asymmetric_key_start is called when entering the alter_asymmetric_key_start production.
	EnterAlter_asymmetric_key_start(c *Alter_asymmetric_key_startContext)

	// EnterAsymmetric_key_option is called when entering the asymmetric_key_option production.
	EnterAsymmetric_key_option(c *Asymmetric_key_optionContext)

	// EnterAsymmetric_key_option_start is called when entering the asymmetric_key_option_start production.
	EnterAsymmetric_key_option_start(c *Asymmetric_key_option_startContext)

	// EnterAsymmetric_key_password_change_option is called when entering the asymmetric_key_password_change_option production.
	EnterAsymmetric_key_password_change_option(c *Asymmetric_key_password_change_optionContext)

	// EnterCreate_asymmetric_key is called when entering the create_asymmetric_key production.
	EnterCreate_asymmetric_key(c *Create_asymmetric_keyContext)

	// EnterDrop_asymmetric_key is called when entering the drop_asymmetric_key production.
	EnterDrop_asymmetric_key(c *Drop_asymmetric_keyContext)

	// EnterAlter_authorization is called when entering the alter_authorization production.
	EnterAlter_authorization(c *Alter_authorizationContext)

	// EnterAuthorization_grantee is called when entering the authorization_grantee production.
	EnterAuthorization_grantee(c *Authorization_granteeContext)

	// EnterEntity_to is called when entering the entity_to production.
	EnterEntity_to(c *Entity_toContext)

	// EnterColon_colon is called when entering the colon_colon production.
	EnterColon_colon(c *Colon_colonContext)

	// EnterAlter_authorization_start is called when entering the alter_authorization_start production.
	EnterAlter_authorization_start(c *Alter_authorization_startContext)

	// EnterAlter_authorization_for_sql_database is called when entering the alter_authorization_for_sql_database production.
	EnterAlter_authorization_for_sql_database(c *Alter_authorization_for_sql_databaseContext)

	// EnterAlter_authorization_for_azure_dw is called when entering the alter_authorization_for_azure_dw production.
	EnterAlter_authorization_for_azure_dw(c *Alter_authorization_for_azure_dwContext)

	// EnterAlter_authorization_for_parallel_dw is called when entering the alter_authorization_for_parallel_dw production.
	EnterAlter_authorization_for_parallel_dw(c *Alter_authorization_for_parallel_dwContext)

	// EnterClass_type is called when entering the class_type production.
	EnterClass_type(c *Class_typeContext)

	// EnterClass_type_for_sql_database is called when entering the class_type_for_sql_database production.
	EnterClass_type_for_sql_database(c *Class_type_for_sql_databaseContext)

	// EnterClass_type_for_azure_dw is called when entering the class_type_for_azure_dw production.
	EnterClass_type_for_azure_dw(c *Class_type_for_azure_dwContext)

	// EnterClass_type_for_parallel_dw is called when entering the class_type_for_parallel_dw production.
	EnterClass_type_for_parallel_dw(c *Class_type_for_parallel_dwContext)

	// EnterClass_type_for_grant is called when entering the class_type_for_grant production.
	EnterClass_type_for_grant(c *Class_type_for_grantContext)

	// EnterDrop_availability_group is called when entering the drop_availability_group production.
	EnterDrop_availability_group(c *Drop_availability_groupContext)

	// EnterAlter_availability_group is called when entering the alter_availability_group production.
	EnterAlter_availability_group(c *Alter_availability_groupContext)

	// EnterAlter_availability_group_start is called when entering the alter_availability_group_start production.
	EnterAlter_availability_group_start(c *Alter_availability_group_startContext)

	// EnterAlter_availability_group_options is called when entering the alter_availability_group_options production.
	EnterAlter_availability_group_options(c *Alter_availability_group_optionsContext)

	// EnterIp_v4_failover is called when entering the ip_v4_failover production.
	EnterIp_v4_failover(c *Ip_v4_failoverContext)

	// EnterIp_v6_failover is called when entering the ip_v6_failover production.
	EnterIp_v6_failover(c *Ip_v6_failoverContext)

	// EnterCreate_or_alter_broker_priority is called when entering the create_or_alter_broker_priority production.
	EnterCreate_or_alter_broker_priority(c *Create_or_alter_broker_priorityContext)

	// EnterDrop_broker_priority is called when entering the drop_broker_priority production.
	EnterDrop_broker_priority(c *Drop_broker_priorityContext)

	// EnterAlter_certificate is called when entering the alter_certificate production.
	EnterAlter_certificate(c *Alter_certificateContext)

	// EnterAlter_column_encryption_key is called when entering the alter_column_encryption_key production.
	EnterAlter_column_encryption_key(c *Alter_column_encryption_keyContext)

	// EnterCreate_column_encryption_key is called when entering the create_column_encryption_key production.
	EnterCreate_column_encryption_key(c *Create_column_encryption_keyContext)

	// EnterDrop_certificate is called when entering the drop_certificate production.
	EnterDrop_certificate(c *Drop_certificateContext)

	// EnterDrop_column_encryption_key is called when entering the drop_column_encryption_key production.
	EnterDrop_column_encryption_key(c *Drop_column_encryption_keyContext)

	// EnterDrop_column_master_key is called when entering the drop_column_master_key production.
	EnterDrop_column_master_key(c *Drop_column_master_keyContext)

	// EnterDrop_contract is called when entering the drop_contract production.
	EnterDrop_contract(c *Drop_contractContext)

	// EnterDrop_credential is called when entering the drop_credential production.
	EnterDrop_credential(c *Drop_credentialContext)

	// EnterDrop_cryptograhic_provider is called when entering the drop_cryptograhic_provider production.
	EnterDrop_cryptograhic_provider(c *Drop_cryptograhic_providerContext)

	// EnterDrop_database is called when entering the drop_database production.
	EnterDrop_database(c *Drop_databaseContext)

	// EnterDrop_database_audit_specification is called when entering the drop_database_audit_specification production.
	EnterDrop_database_audit_specification(c *Drop_database_audit_specificationContext)

	// EnterDrop_database_encryption_key is called when entering the drop_database_encryption_key production.
	EnterDrop_database_encryption_key(c *Drop_database_encryption_keyContext)

	// EnterDrop_database_scoped_credential is called when entering the drop_database_scoped_credential production.
	EnterDrop_database_scoped_credential(c *Drop_database_scoped_credentialContext)

	// EnterDrop_default is called when entering the drop_default production.
	EnterDrop_default(c *Drop_defaultContext)

	// EnterDrop_endpoint is called when entering the drop_endpoint production.
	EnterDrop_endpoint(c *Drop_endpointContext)

	// EnterDrop_external_data_source is called when entering the drop_external_data_source production.
	EnterDrop_external_data_source(c *Drop_external_data_sourceContext)

	// EnterDrop_external_file_format is called when entering the drop_external_file_format production.
	EnterDrop_external_file_format(c *Drop_external_file_formatContext)

	// EnterDrop_external_library is called when entering the drop_external_library production.
	EnterDrop_external_library(c *Drop_external_libraryContext)

	// EnterDrop_external_resource_pool is called when entering the drop_external_resource_pool production.
	EnterDrop_external_resource_pool(c *Drop_external_resource_poolContext)

	// EnterDrop_external_table is called when entering the drop_external_table production.
	EnterDrop_external_table(c *Drop_external_tableContext)

	// EnterDrop_event_notifications is called when entering the drop_event_notifications production.
	EnterDrop_event_notifications(c *Drop_event_notificationsContext)

	// EnterDrop_event_session is called when entering the drop_event_session production.
	EnterDrop_event_session(c *Drop_event_sessionContext)

	// EnterDrop_fulltext_catalog is called when entering the drop_fulltext_catalog production.
	EnterDrop_fulltext_catalog(c *Drop_fulltext_catalogContext)

	// EnterDrop_fulltext_index is called when entering the drop_fulltext_index production.
	EnterDrop_fulltext_index(c *Drop_fulltext_indexContext)

	// EnterDrop_fulltext_stoplist is called when entering the drop_fulltext_stoplist production.
	EnterDrop_fulltext_stoplist(c *Drop_fulltext_stoplistContext)

	// EnterDrop_login is called when entering the drop_login production.
	EnterDrop_login(c *Drop_loginContext)

	// EnterDrop_master_key is called when entering the drop_master_key production.
	EnterDrop_master_key(c *Drop_master_keyContext)

	// EnterDrop_message_type is called when entering the drop_message_type production.
	EnterDrop_message_type(c *Drop_message_typeContext)

	// EnterDrop_partition_function is called when entering the drop_partition_function production.
	EnterDrop_partition_function(c *Drop_partition_functionContext)

	// EnterDrop_partition_scheme is called when entering the drop_partition_scheme production.
	EnterDrop_partition_scheme(c *Drop_partition_schemeContext)

	// EnterDrop_queue is called when entering the drop_queue production.
	EnterDrop_queue(c *Drop_queueContext)

	// EnterDrop_remote_service_binding is called when entering the drop_remote_service_binding production.
	EnterDrop_remote_service_binding(c *Drop_remote_service_bindingContext)

	// EnterDrop_resource_pool is called when entering the drop_resource_pool production.
	EnterDrop_resource_pool(c *Drop_resource_poolContext)

	// EnterDrop_db_role is called when entering the drop_db_role production.
	EnterDrop_db_role(c *Drop_db_roleContext)

	// EnterDrop_route is called when entering the drop_route production.
	EnterDrop_route(c *Drop_routeContext)

	// EnterDrop_rule is called when entering the drop_rule production.
	EnterDrop_rule(c *Drop_ruleContext)

	// EnterDrop_schema is called when entering the drop_schema production.
	EnterDrop_schema(c *Drop_schemaContext)

	// EnterDrop_search_property_list is called when entering the drop_search_property_list production.
	EnterDrop_search_property_list(c *Drop_search_property_listContext)

	// EnterDrop_security_policy is called when entering the drop_security_policy production.
	EnterDrop_security_policy(c *Drop_security_policyContext)

	// EnterDrop_sequence is called when entering the drop_sequence production.
	EnterDrop_sequence(c *Drop_sequenceContext)

	// EnterDrop_server_audit is called when entering the drop_server_audit production.
	EnterDrop_server_audit(c *Drop_server_auditContext)

	// EnterDrop_server_audit_specification is called when entering the drop_server_audit_specification production.
	EnterDrop_server_audit_specification(c *Drop_server_audit_specificationContext)

	// EnterDrop_server_role is called when entering the drop_server_role production.
	EnterDrop_server_role(c *Drop_server_roleContext)

	// EnterDrop_service is called when entering the drop_service production.
	EnterDrop_service(c *Drop_serviceContext)

	// EnterDrop_signature is called when entering the drop_signature production.
	EnterDrop_signature(c *Drop_signatureContext)

	// EnterDrop_statistics_name_azure_dw_and_pdw is called when entering the drop_statistics_name_azure_dw_and_pdw production.
	EnterDrop_statistics_name_azure_dw_and_pdw(c *Drop_statistics_name_azure_dw_and_pdwContext)

	// EnterDrop_symmetric_key is called when entering the drop_symmetric_key production.
	EnterDrop_symmetric_key(c *Drop_symmetric_keyContext)

	// EnterDrop_synonym is called when entering the drop_synonym production.
	EnterDrop_synonym(c *Drop_synonymContext)

	// EnterDrop_user is called when entering the drop_user production.
	EnterDrop_user(c *Drop_userContext)

	// EnterDrop_workload_group is called when entering the drop_workload_group production.
	EnterDrop_workload_group(c *Drop_workload_groupContext)

	// EnterDrop_xml_schema_collection is called when entering the drop_xml_schema_collection production.
	EnterDrop_xml_schema_collection(c *Drop_xml_schema_collectionContext)

	// EnterDisable_trigger is called when entering the disable_trigger production.
	EnterDisable_trigger(c *Disable_triggerContext)

	// EnterEnable_trigger is called when entering the enable_trigger production.
	EnterEnable_trigger(c *Enable_triggerContext)

	// EnterLock_table is called when entering the lock_table production.
	EnterLock_table(c *Lock_tableContext)

	// EnterTruncate_table is called when entering the truncate_table production.
	EnterTruncate_table(c *Truncate_tableContext)

	// EnterCreate_column_master_key is called when entering the create_column_master_key production.
	EnterCreate_column_master_key(c *Create_column_master_keyContext)

	// EnterAlter_credential is called when entering the alter_credential production.
	EnterAlter_credential(c *Alter_credentialContext)

	// EnterCreate_credential is called when entering the create_credential production.
	EnterCreate_credential(c *Create_credentialContext)

	// EnterAlter_cryptographic_provider is called when entering the alter_cryptographic_provider production.
	EnterAlter_cryptographic_provider(c *Alter_cryptographic_providerContext)

	// EnterCreate_cryptographic_provider is called when entering the create_cryptographic_provider production.
	EnterCreate_cryptographic_provider(c *Create_cryptographic_providerContext)

	// EnterCreate_endpoint is called when entering the create_endpoint production.
	EnterCreate_endpoint(c *Create_endpointContext)

	// EnterEndpoint_encryption_alogorithm_clause is called when entering the endpoint_encryption_alogorithm_clause production.
	EnterEndpoint_encryption_alogorithm_clause(c *Endpoint_encryption_alogorithm_clauseContext)

	// EnterEndpoint_authentication_clause is called when entering the endpoint_authentication_clause production.
	EnterEndpoint_authentication_clause(c *Endpoint_authentication_clauseContext)

	// EnterEndpoint_listener_clause is called when entering the endpoint_listener_clause production.
	EnterEndpoint_listener_clause(c *Endpoint_listener_clauseContext)

	// EnterCreate_event_notification is called when entering the create_event_notification production.
	EnterCreate_event_notification(c *Create_event_notificationContext)

	// EnterCreate_or_alter_event_session is called when entering the create_or_alter_event_session production.
	EnterCreate_or_alter_event_session(c *Create_or_alter_event_sessionContext)

	// EnterEvent_session_predicate_expression is called when entering the event_session_predicate_expression production.
	EnterEvent_session_predicate_expression(c *Event_session_predicate_expressionContext)

	// EnterEvent_session_predicate_factor is called when entering the event_session_predicate_factor production.
	EnterEvent_session_predicate_factor(c *Event_session_predicate_factorContext)

	// EnterEvent_session_predicate_leaf is called when entering the event_session_predicate_leaf production.
	EnterEvent_session_predicate_leaf(c *Event_session_predicate_leafContext)

	// EnterAlter_external_data_source is called when entering the alter_external_data_source production.
	EnterAlter_external_data_source(c *Alter_external_data_sourceContext)

	// EnterAlter_external_library is called when entering the alter_external_library production.
	EnterAlter_external_library(c *Alter_external_libraryContext)

	// EnterCreate_external_library is called when entering the create_external_library production.
	EnterCreate_external_library(c *Create_external_libraryContext)

	// EnterAlter_external_resource_pool is called when entering the alter_external_resource_pool production.
	EnterAlter_external_resource_pool(c *Alter_external_resource_poolContext)

	// EnterCreate_external_resource_pool is called when entering the create_external_resource_pool production.
	EnterCreate_external_resource_pool(c *Create_external_resource_poolContext)

	// EnterAlter_fulltext_catalog is called when entering the alter_fulltext_catalog production.
	EnterAlter_fulltext_catalog(c *Alter_fulltext_catalogContext)

	// EnterCreate_fulltext_catalog is called when entering the create_fulltext_catalog production.
	EnterCreate_fulltext_catalog(c *Create_fulltext_catalogContext)

	// EnterAlter_fulltext_stoplist is called when entering the alter_fulltext_stoplist production.
	EnterAlter_fulltext_stoplist(c *Alter_fulltext_stoplistContext)

	// EnterCreate_fulltext_stoplist is called when entering the create_fulltext_stoplist production.
	EnterCreate_fulltext_stoplist(c *Create_fulltext_stoplistContext)

	// EnterAlter_login_sql_server is called when entering the alter_login_sql_server production.
	EnterAlter_login_sql_server(c *Alter_login_sql_serverContext)

	// EnterCreate_login_sql_server is called when entering the create_login_sql_server production.
	EnterCreate_login_sql_server(c *Create_login_sql_serverContext)

	// EnterAlter_login_azure_sql is called when entering the alter_login_azure_sql production.
	EnterAlter_login_azure_sql(c *Alter_login_azure_sqlContext)

	// EnterCreate_login_azure_sql is called when entering the create_login_azure_sql production.
	EnterCreate_login_azure_sql(c *Create_login_azure_sqlContext)

	// EnterAlter_login_azure_sql_dw_and_pdw is called when entering the alter_login_azure_sql_dw_and_pdw production.
	EnterAlter_login_azure_sql_dw_and_pdw(c *Alter_login_azure_sql_dw_and_pdwContext)

	// EnterCreate_login_pdw is called when entering the create_login_pdw production.
	EnterCreate_login_pdw(c *Create_login_pdwContext)

	// EnterAlter_master_key_sql_server is called when entering the alter_master_key_sql_server production.
	EnterAlter_master_key_sql_server(c *Alter_master_key_sql_serverContext)

	// EnterCreate_master_key_sql_server is called when entering the create_master_key_sql_server production.
	EnterCreate_master_key_sql_server(c *Create_master_key_sql_serverContext)

	// EnterAlter_master_key_azure_sql is called when entering the alter_master_key_azure_sql production.
	EnterAlter_master_key_azure_sql(c *Alter_master_key_azure_sqlContext)

	// EnterCreate_master_key_azure_sql is called when entering the create_master_key_azure_sql production.
	EnterCreate_master_key_azure_sql(c *Create_master_key_azure_sqlContext)

	// EnterAlter_message_type is called when entering the alter_message_type production.
	EnterAlter_message_type(c *Alter_message_typeContext)

	// EnterAlter_partition_function is called when entering the alter_partition_function production.
	EnterAlter_partition_function(c *Alter_partition_functionContext)

	// EnterAlter_partition_scheme is called when entering the alter_partition_scheme production.
	EnterAlter_partition_scheme(c *Alter_partition_schemeContext)

	// EnterAlter_remote_service_binding is called when entering the alter_remote_service_binding production.
	EnterAlter_remote_service_binding(c *Alter_remote_service_bindingContext)

	// EnterCreate_remote_service_binding is called when entering the create_remote_service_binding production.
	EnterCreate_remote_service_binding(c *Create_remote_service_bindingContext)

	// EnterCreate_resource_pool is called when entering the create_resource_pool production.
	EnterCreate_resource_pool(c *Create_resource_poolContext)

	// EnterAlter_resource_governor is called when entering the alter_resource_governor production.
	EnterAlter_resource_governor(c *Alter_resource_governorContext)

	// EnterAlter_database_audit_specification is called when entering the alter_database_audit_specification production.
	EnterAlter_database_audit_specification(c *Alter_database_audit_specificationContext)

	// EnterAudit_action_spec_group is called when entering the audit_action_spec_group production.
	EnterAudit_action_spec_group(c *Audit_action_spec_groupContext)

	// EnterAudit_action_specification is called when entering the audit_action_specification production.
	EnterAudit_action_specification(c *Audit_action_specificationContext)

	// EnterAction_specification is called when entering the action_specification production.
	EnterAction_specification(c *Action_specificationContext)

	// EnterAudit_class_name is called when entering the audit_class_name production.
	EnterAudit_class_name(c *Audit_class_nameContext)

	// EnterAudit_securable is called when entering the audit_securable production.
	EnterAudit_securable(c *Audit_securableContext)

	// EnterAlter_db_role is called when entering the alter_db_role production.
	EnterAlter_db_role(c *Alter_db_roleContext)

	// EnterCreate_database_audit_specification is called when entering the create_database_audit_specification production.
	EnterCreate_database_audit_specification(c *Create_database_audit_specificationContext)

	// EnterCreate_db_role is called when entering the create_db_role production.
	EnterCreate_db_role(c *Create_db_roleContext)

	// EnterCreate_route is called when entering the create_route production.
	EnterCreate_route(c *Create_routeContext)

	// EnterCreate_rule is called when entering the create_rule production.
	EnterCreate_rule(c *Create_ruleContext)

	// EnterAlter_schema_sql is called when entering the alter_schema_sql production.
	EnterAlter_schema_sql(c *Alter_schema_sqlContext)

	// EnterCreate_schema is called when entering the create_schema production.
	EnterCreate_schema(c *Create_schemaContext)

	// EnterCreate_schema_azure_sql_dw_and_pdw is called when entering the create_schema_azure_sql_dw_and_pdw production.
	EnterCreate_schema_azure_sql_dw_and_pdw(c *Create_schema_azure_sql_dw_and_pdwContext)

	// EnterAlter_schema_azure_sql_dw_and_pdw is called when entering the alter_schema_azure_sql_dw_and_pdw production.
	EnterAlter_schema_azure_sql_dw_and_pdw(c *Alter_schema_azure_sql_dw_and_pdwContext)

	// EnterCreate_search_property_list is called when entering the create_search_property_list production.
	EnterCreate_search_property_list(c *Create_search_property_listContext)

	// EnterCreate_security_policy is called when entering the create_security_policy production.
	EnterCreate_security_policy(c *Create_security_policyContext)

	// EnterAlter_sequence is called when entering the alter_sequence production.
	EnterAlter_sequence(c *Alter_sequenceContext)

	// EnterCreate_sequence is called when entering the create_sequence production.
	EnterCreate_sequence(c *Create_sequenceContext)

	// EnterAlter_server_audit is called when entering the alter_server_audit production.
	EnterAlter_server_audit(c *Alter_server_auditContext)

	// EnterCreate_server_audit is called when entering the create_server_audit production.
	EnterCreate_server_audit(c *Create_server_auditContext)

	// EnterAlter_server_audit_specification is called when entering the alter_server_audit_specification production.
	EnterAlter_server_audit_specification(c *Alter_server_audit_specificationContext)

	// EnterCreate_server_audit_specification is called when entering the create_server_audit_specification production.
	EnterCreate_server_audit_specification(c *Create_server_audit_specificationContext)

	// EnterAlter_server_configuration is called when entering the alter_server_configuration production.
	EnterAlter_server_configuration(c *Alter_server_configurationContext)

	// EnterAlter_server_role is called when entering the alter_server_role production.
	EnterAlter_server_role(c *Alter_server_roleContext)

	// EnterCreate_server_role is called when entering the create_server_role production.
	EnterCreate_server_role(c *Create_server_roleContext)

	// EnterAlter_server_role_pdw is called when entering the alter_server_role_pdw production.
	EnterAlter_server_role_pdw(c *Alter_server_role_pdwContext)

	// EnterAlter_service is called when entering the alter_service production.
	EnterAlter_service(c *Alter_serviceContext)

	// EnterOpt_arg_clause is called when entering the opt_arg_clause production.
	EnterOpt_arg_clause(c *Opt_arg_clauseContext)

	// EnterCreate_service is called when entering the create_service production.
	EnterCreate_service(c *Create_serviceContext)

	// EnterAlter_service_master_key is called when entering the alter_service_master_key production.
	EnterAlter_service_master_key(c *Alter_service_master_keyContext)

	// EnterAlter_symmetric_key is called when entering the alter_symmetric_key production.
	EnterAlter_symmetric_key(c *Alter_symmetric_keyContext)

	// EnterCreate_synonym is called when entering the create_synonym production.
	EnterCreate_synonym(c *Create_synonymContext)

	// EnterAlter_user is called when entering the alter_user production.
	EnterAlter_user(c *Alter_userContext)

	// EnterCreate_user is called when entering the create_user production.
	EnterCreate_user(c *Create_userContext)

	// EnterCreate_user_azure_sql_dw is called when entering the create_user_azure_sql_dw production.
	EnterCreate_user_azure_sql_dw(c *Create_user_azure_sql_dwContext)

	// EnterAlter_user_azure_sql is called when entering the alter_user_azure_sql production.
	EnterAlter_user_azure_sql(c *Alter_user_azure_sqlContext)

	// EnterAlter_workload_group is called when entering the alter_workload_group production.
	EnterAlter_workload_group(c *Alter_workload_groupContext)

	// EnterCreate_workload_group is called when entering the create_workload_group production.
	EnterCreate_workload_group(c *Create_workload_groupContext)

	// EnterCreate_xml_schema_collection is called when entering the create_xml_schema_collection production.
	EnterCreate_xml_schema_collection(c *Create_xml_schema_collectionContext)

	// EnterCreate_partition_function is called when entering the create_partition_function production.
	EnterCreate_partition_function(c *Create_partition_functionContext)

	// EnterCreate_partition_scheme is called when entering the create_partition_scheme production.
	EnterCreate_partition_scheme(c *Create_partition_schemeContext)

	// EnterCreate_queue is called when entering the create_queue production.
	EnterCreate_queue(c *Create_queueContext)

	// EnterQueue_settings is called when entering the queue_settings production.
	EnterQueue_settings(c *Queue_settingsContext)

	// EnterAlter_queue is called when entering the alter_queue production.
	EnterAlter_queue(c *Alter_queueContext)

	// EnterQueue_action is called when entering the queue_action production.
	EnterQueue_action(c *Queue_actionContext)

	// EnterQueue_rebuild_options is called when entering the queue_rebuild_options production.
	EnterQueue_rebuild_options(c *Queue_rebuild_optionsContext)

	// EnterCreate_contract is called when entering the create_contract production.
	EnterCreate_contract(c *Create_contractContext)

	// EnterConversation_statement is called when entering the conversation_statement production.
	EnterConversation_statement(c *Conversation_statementContext)

	// EnterMessage_statement is called when entering the message_statement production.
	EnterMessage_statement(c *Message_statementContext)

	// EnterMerge_statement is called when entering the merge_statement production.
	EnterMerge_statement(c *Merge_statementContext)

	// EnterWhen_matches is called when entering the when_matches production.
	EnterWhen_matches(c *When_matchesContext)

	// EnterMerge_matched is called when entering the merge_matched production.
	EnterMerge_matched(c *Merge_matchedContext)

	// EnterMerge_not_matched is called when entering the merge_not_matched production.
	EnterMerge_not_matched(c *Merge_not_matchedContext)

	// EnterDelete_statement is called when entering the delete_statement production.
	EnterDelete_statement(c *Delete_statementContext)

	// EnterDelete_statement_from is called when entering the delete_statement_from production.
	EnterDelete_statement_from(c *Delete_statement_fromContext)

	// EnterInsert_statement is called when entering the insert_statement production.
	EnterInsert_statement(c *Insert_statementContext)

	// EnterInsert_statement_value is called when entering the insert_statement_value production.
	EnterInsert_statement_value(c *Insert_statement_valueContext)

	// EnterReceive_statement is called when entering the receive_statement production.
	EnterReceive_statement(c *Receive_statementContext)

	// EnterSelect_statement_standalone is called when entering the select_statement_standalone production.
	EnterSelect_statement_standalone(c *Select_statement_standaloneContext)

	// EnterSelect_statement is called when entering the select_statement production.
	EnterSelect_statement(c *Select_statementContext)

	// EnterTime is called when entering the time production.
	EnterTime(c *TimeContext)

	// EnterUpdate_statement is called when entering the update_statement production.
	EnterUpdate_statement(c *Update_statementContext)

	// EnterOutput_clause is called when entering the output_clause production.
	EnterOutput_clause(c *Output_clauseContext)

	// EnterOutput_dml_list_elem is called when entering the output_dml_list_elem production.
	EnterOutput_dml_list_elem(c *Output_dml_list_elemContext)

	// EnterCreate_database is called when entering the create_database production.
	EnterCreate_database(c *Create_databaseContext)

	// EnterCreate_index is called when entering the create_index production.
	EnterCreate_index(c *Create_indexContext)

	// EnterCreate_index_options is called when entering the create_index_options production.
	EnterCreate_index_options(c *Create_index_optionsContext)

	// EnterRelational_index_option is called when entering the relational_index_option production.
	EnterRelational_index_option(c *Relational_index_optionContext)

	// EnterAlter_index is called when entering the alter_index production.
	EnterAlter_index(c *Alter_indexContext)

	// EnterResumable_index_options is called when entering the resumable_index_options production.
	EnterResumable_index_options(c *Resumable_index_optionsContext)

	// EnterResumable_index_option is called when entering the resumable_index_option production.
	EnterResumable_index_option(c *Resumable_index_optionContext)

	// EnterReorganize_partition is called when entering the reorganize_partition production.
	EnterReorganize_partition(c *Reorganize_partitionContext)

	// EnterReorganize_options is called when entering the reorganize_options production.
	EnterReorganize_options(c *Reorganize_optionsContext)

	// EnterReorganize_option is called when entering the reorganize_option production.
	EnterReorganize_option(c *Reorganize_optionContext)

	// EnterSet_index_options is called when entering the set_index_options production.
	EnterSet_index_options(c *Set_index_optionsContext)

	// EnterSet_index_option is called when entering the set_index_option production.
	EnterSet_index_option(c *Set_index_optionContext)

	// EnterRebuild_partition is called when entering the rebuild_partition production.
	EnterRebuild_partition(c *Rebuild_partitionContext)

	// EnterRebuild_index_options is called when entering the rebuild_index_options production.
	EnterRebuild_index_options(c *Rebuild_index_optionsContext)

	// EnterRebuild_index_option is called when entering the rebuild_index_option production.
	EnterRebuild_index_option(c *Rebuild_index_optionContext)

	// EnterSingle_partition_rebuild_index_options is called when entering the single_partition_rebuild_index_options production.
	EnterSingle_partition_rebuild_index_options(c *Single_partition_rebuild_index_optionsContext)

	// EnterSingle_partition_rebuild_index_option is called when entering the single_partition_rebuild_index_option production.
	EnterSingle_partition_rebuild_index_option(c *Single_partition_rebuild_index_optionContext)

	// EnterOn_partitions is called when entering the on_partitions production.
	EnterOn_partitions(c *On_partitionsContext)

	// EnterCreate_columnstore_index is called when entering the create_columnstore_index production.
	EnterCreate_columnstore_index(c *Create_columnstore_indexContext)

	// EnterCreate_columnstore_index_options is called when entering the create_columnstore_index_options production.
	EnterCreate_columnstore_index_options(c *Create_columnstore_index_optionsContext)

	// EnterColumnstore_index_option is called when entering the columnstore_index_option production.
	EnterColumnstore_index_option(c *Columnstore_index_optionContext)

	// EnterCreate_nonclustered_columnstore_index is called when entering the create_nonclustered_columnstore_index production.
	EnterCreate_nonclustered_columnstore_index(c *Create_nonclustered_columnstore_indexContext)

	// EnterCreate_xml_index is called when entering the create_xml_index production.
	EnterCreate_xml_index(c *Create_xml_indexContext)

	// EnterXml_index_options is called when entering the xml_index_options production.
	EnterXml_index_options(c *Xml_index_optionsContext)

	// EnterXml_index_option is called when entering the xml_index_option production.
	EnterXml_index_option(c *Xml_index_optionContext)

	// EnterCreate_or_alter_procedure is called when entering the create_or_alter_procedure production.
	EnterCreate_or_alter_procedure(c *Create_or_alter_procedureContext)

	// EnterAs_external_name is called when entering the as_external_name production.
	EnterAs_external_name(c *As_external_nameContext)

	// EnterCreate_or_alter_trigger is called when entering the create_or_alter_trigger production.
	EnterCreate_or_alter_trigger(c *Create_or_alter_triggerContext)

	// EnterCreate_or_alter_dml_trigger is called when entering the create_or_alter_dml_trigger production.
	EnterCreate_or_alter_dml_trigger(c *Create_or_alter_dml_triggerContext)

	// EnterDml_trigger_option is called when entering the dml_trigger_option production.
	EnterDml_trigger_option(c *Dml_trigger_optionContext)

	// EnterDml_trigger_operation is called when entering the dml_trigger_operation production.
	EnterDml_trigger_operation(c *Dml_trigger_operationContext)

	// EnterCreate_or_alter_ddl_trigger is called when entering the create_or_alter_ddl_trigger production.
	EnterCreate_or_alter_ddl_trigger(c *Create_or_alter_ddl_triggerContext)

	// EnterDdl_trigger_operation is called when entering the ddl_trigger_operation production.
	EnterDdl_trigger_operation(c *Ddl_trigger_operationContext)

	// EnterCreate_or_alter_function is called when entering the create_or_alter_function production.
	EnterCreate_or_alter_function(c *Create_or_alter_functionContext)

	// EnterFunc_body_returns_select is called when entering the func_body_returns_select production.
	EnterFunc_body_returns_select(c *Func_body_returns_selectContext)

	// EnterFunc_body_returns_table is called when entering the func_body_returns_table production.
	EnterFunc_body_returns_table(c *Func_body_returns_tableContext)

	// EnterFunc_body_returns_scalar is called when entering the func_body_returns_scalar production.
	EnterFunc_body_returns_scalar(c *Func_body_returns_scalarContext)

	// EnterProcedure_param_default_value is called when entering the procedure_param_default_value production.
	EnterProcedure_param_default_value(c *Procedure_param_default_valueContext)

	// EnterProcedure_param is called when entering the procedure_param production.
	EnterProcedure_param(c *Procedure_paramContext)

	// EnterProcedure_option is called when entering the procedure_option production.
	EnterProcedure_option(c *Procedure_optionContext)

	// EnterFunction_option is called when entering the function_option production.
	EnterFunction_option(c *Function_optionContext)

	// EnterCreate_statistics is called when entering the create_statistics production.
	EnterCreate_statistics(c *Create_statisticsContext)

	// EnterUpdate_statistics is called when entering the update_statistics production.
	EnterUpdate_statistics(c *Update_statisticsContext)

	// EnterUpdate_statistics_options is called when entering the update_statistics_options production.
	EnterUpdate_statistics_options(c *Update_statistics_optionsContext)

	// EnterUpdate_statistics_option is called when entering the update_statistics_option production.
	EnterUpdate_statistics_option(c *Update_statistics_optionContext)

	// EnterCreate_table is called when entering the create_table production.
	EnterCreate_table(c *Create_tableContext)

	// EnterTable_indices is called when entering the table_indices production.
	EnterTable_indices(c *Table_indicesContext)

	// EnterTable_options is called when entering the table_options production.
	EnterTable_options(c *Table_optionsContext)

	// EnterTable_option is called when entering the table_option production.
	EnterTable_option(c *Table_optionContext)

	// EnterCreate_table_index_options is called when entering the create_table_index_options production.
	EnterCreate_table_index_options(c *Create_table_index_optionsContext)

	// EnterCreate_table_index_option is called when entering the create_table_index_option production.
	EnterCreate_table_index_option(c *Create_table_index_optionContext)

	// EnterCreate_view is called when entering the create_view production.
	EnterCreate_view(c *Create_viewContext)

	// EnterView_attribute is called when entering the view_attribute production.
	EnterView_attribute(c *View_attributeContext)

	// EnterAlter_table is called when entering the alter_table production.
	EnterAlter_table(c *Alter_tableContext)

	// EnterSwitch_partition is called when entering the switch_partition production.
	EnterSwitch_partition(c *Switch_partitionContext)

	// EnterLow_priority_lock_wait is called when entering the low_priority_lock_wait production.
	EnterLow_priority_lock_wait(c *Low_priority_lock_waitContext)

	// EnterAlter_database is called when entering the alter_database production.
	EnterAlter_database(c *Alter_databaseContext)

	// EnterAdd_or_modify_files is called when entering the add_or_modify_files production.
	EnterAdd_or_modify_files(c *Add_or_modify_filesContext)

	// EnterFilespec is called when entering the filespec production.
	EnterFilespec(c *FilespecContext)

	// EnterAdd_or_modify_filegroups is called when entering the add_or_modify_filegroups production.
	EnterAdd_or_modify_filegroups(c *Add_or_modify_filegroupsContext)

	// EnterFilegroup_updatability_option is called when entering the filegroup_updatability_option production.
	EnterFilegroup_updatability_option(c *Filegroup_updatability_optionContext)

	// EnterDatabase_optionspec is called when entering the database_optionspec production.
	EnterDatabase_optionspec(c *Database_optionspecContext)

	// EnterAuto_option is called when entering the auto_option production.
	EnterAuto_option(c *Auto_optionContext)

	// EnterChange_tracking_option is called when entering the change_tracking_option production.
	EnterChange_tracking_option(c *Change_tracking_optionContext)

	// EnterChange_tracking_option_list is called when entering the change_tracking_option_list production.
	EnterChange_tracking_option_list(c *Change_tracking_option_listContext)

	// EnterContainment_option is called when entering the containment_option production.
	EnterContainment_option(c *Containment_optionContext)

	// EnterCursor_option is called when entering the cursor_option production.
	EnterCursor_option(c *Cursor_optionContext)

	// EnterAlter_endpoint is called when entering the alter_endpoint production.
	EnterAlter_endpoint(c *Alter_endpointContext)

	// EnterDatabase_mirroring_option is called when entering the database_mirroring_option production.
	EnterDatabase_mirroring_option(c *Database_mirroring_optionContext)

	// EnterMirroring_set_option is called when entering the mirroring_set_option production.
	EnterMirroring_set_option(c *Mirroring_set_optionContext)

	// EnterMirroring_partner is called when entering the mirroring_partner production.
	EnterMirroring_partner(c *Mirroring_partnerContext)

	// EnterMirroring_witness is called when entering the mirroring_witness production.
	EnterMirroring_witness(c *Mirroring_witnessContext)

	// EnterWitness_partner_equal is called when entering the witness_partner_equal production.
	EnterWitness_partner_equal(c *Witness_partner_equalContext)

	// EnterPartner_option is called when entering the partner_option production.
	EnterPartner_option(c *Partner_optionContext)

	// EnterWitness_option is called when entering the witness_option production.
	EnterWitness_option(c *Witness_optionContext)

	// EnterWitness_server is called when entering the witness_server production.
	EnterWitness_server(c *Witness_serverContext)

	// EnterPartner_server is called when entering the partner_server production.
	EnterPartner_server(c *Partner_serverContext)

	// EnterMirroring_host_port_seperator is called when entering the mirroring_host_port_seperator production.
	EnterMirroring_host_port_seperator(c *Mirroring_host_port_seperatorContext)

	// EnterPartner_server_tcp_prefix is called when entering the partner_server_tcp_prefix production.
	EnterPartner_server_tcp_prefix(c *Partner_server_tcp_prefixContext)

	// EnterPort_number is called when entering the port_number production.
	EnterPort_number(c *Port_numberContext)

	// EnterHost is called when entering the host production.
	EnterHost(c *HostContext)

	// EnterDate_correlation_optimization_option is called when entering the date_correlation_optimization_option production.
	EnterDate_correlation_optimization_option(c *Date_correlation_optimization_optionContext)

	// EnterDb_encryption_option is called when entering the db_encryption_option production.
	EnterDb_encryption_option(c *Db_encryption_optionContext)

	// EnterDb_state_option is called when entering the db_state_option production.
	EnterDb_state_option(c *Db_state_optionContext)

	// EnterDb_update_option is called when entering the db_update_option production.
	EnterDb_update_option(c *Db_update_optionContext)

	// EnterDb_user_access_option is called when entering the db_user_access_option production.
	EnterDb_user_access_option(c *Db_user_access_optionContext)

	// EnterDelayed_durability_option is called when entering the delayed_durability_option production.
	EnterDelayed_durability_option(c *Delayed_durability_optionContext)

	// EnterExternal_access_option is called when entering the external_access_option production.
	EnterExternal_access_option(c *External_access_optionContext)

	// EnterHadr_options is called when entering the hadr_options production.
	EnterHadr_options(c *Hadr_optionsContext)

	// EnterMixed_page_allocation_option is called when entering the mixed_page_allocation_option production.
	EnterMixed_page_allocation_option(c *Mixed_page_allocation_optionContext)

	// EnterParameterization_option is called when entering the parameterization_option production.
	EnterParameterization_option(c *Parameterization_optionContext)

	// EnterRecovery_option is called when entering the recovery_option production.
	EnterRecovery_option(c *Recovery_optionContext)

	// EnterService_broker_option is called when entering the service_broker_option production.
	EnterService_broker_option(c *Service_broker_optionContext)

	// EnterSnapshot_option is called when entering the snapshot_option production.
	EnterSnapshot_option(c *Snapshot_optionContext)

	// EnterSql_option is called when entering the sql_option production.
	EnterSql_option(c *Sql_optionContext)

	// EnterTarget_recovery_time_option is called when entering the target_recovery_time_option production.
	EnterTarget_recovery_time_option(c *Target_recovery_time_optionContext)

	// EnterTermination is called when entering the termination production.
	EnterTermination(c *TerminationContext)

	// EnterDrop_index is called when entering the drop_index production.
	EnterDrop_index(c *Drop_indexContext)

	// EnterDrop_relational_or_xml_or_spatial_index is called when entering the drop_relational_or_xml_or_spatial_index production.
	EnterDrop_relational_or_xml_or_spatial_index(c *Drop_relational_or_xml_or_spatial_indexContext)

	// EnterDrop_backward_compatible_index is called when entering the drop_backward_compatible_index production.
	EnterDrop_backward_compatible_index(c *Drop_backward_compatible_indexContext)

	// EnterDrop_procedure is called when entering the drop_procedure production.
	EnterDrop_procedure(c *Drop_procedureContext)

	// EnterDrop_trigger is called when entering the drop_trigger production.
	EnterDrop_trigger(c *Drop_triggerContext)

	// EnterDrop_dml_trigger is called when entering the drop_dml_trigger production.
	EnterDrop_dml_trigger(c *Drop_dml_triggerContext)

	// EnterDrop_ddl_trigger is called when entering the drop_ddl_trigger production.
	EnterDrop_ddl_trigger(c *Drop_ddl_triggerContext)

	// EnterDrop_function is called when entering the drop_function production.
	EnterDrop_function(c *Drop_functionContext)

	// EnterDrop_statistics is called when entering the drop_statistics production.
	EnterDrop_statistics(c *Drop_statisticsContext)

	// EnterDrop_table is called when entering the drop_table production.
	EnterDrop_table(c *Drop_tableContext)

	// EnterDrop_view is called when entering the drop_view production.
	EnterDrop_view(c *Drop_viewContext)

	// EnterCreate_type is called when entering the create_type production.
	EnterCreate_type(c *Create_typeContext)

	// EnterDrop_type is called when entering the drop_type production.
	EnterDrop_type(c *Drop_typeContext)

	// EnterRowset_function_limited is called when entering the rowset_function_limited production.
	EnterRowset_function_limited(c *Rowset_function_limitedContext)

	// EnterOpenquery is called when entering the openquery production.
	EnterOpenquery(c *OpenqueryContext)

	// EnterOpendatasource is called when entering the opendatasource production.
	EnterOpendatasource(c *OpendatasourceContext)

	// EnterDeclare_statement is called when entering the declare_statement production.
	EnterDeclare_statement(c *Declare_statementContext)

	// EnterXml_declaration is called when entering the xml_declaration production.
	EnterXml_declaration(c *Xml_declarationContext)

	// EnterCursor_statement is called when entering the cursor_statement production.
	EnterCursor_statement(c *Cursor_statementContext)

	// EnterBackup_database is called when entering the backup_database production.
	EnterBackup_database(c *Backup_databaseContext)

	// EnterBackup_log is called when entering the backup_log production.
	EnterBackup_log(c *Backup_logContext)

	// EnterBackup_certificate is called when entering the backup_certificate production.
	EnterBackup_certificate(c *Backup_certificateContext)

	// EnterBackup_master_key is called when entering the backup_master_key production.
	EnterBackup_master_key(c *Backup_master_keyContext)

	// EnterBackup_service_master_key is called when entering the backup_service_master_key production.
	EnterBackup_service_master_key(c *Backup_service_master_keyContext)

	// EnterKill_statement is called when entering the kill_statement production.
	EnterKill_statement(c *Kill_statementContext)

	// EnterKill_process is called when entering the kill_process production.
	EnterKill_process(c *Kill_processContext)

	// EnterKill_query_notification is called when entering the kill_query_notification production.
	EnterKill_query_notification(c *Kill_query_notificationContext)

	// EnterKill_stats_job is called when entering the kill_stats_job production.
	EnterKill_stats_job(c *Kill_stats_jobContext)

	// EnterExecute_statement is called when entering the execute_statement production.
	EnterExecute_statement(c *Execute_statementContext)

	// EnterExecute_body_batch is called when entering the execute_body_batch production.
	EnterExecute_body_batch(c *Execute_body_batchContext)

	// EnterExecute_body is called when entering the execute_body production.
	EnterExecute_body(c *Execute_bodyContext)

	// EnterExecute_statement_arg is called when entering the execute_statement_arg production.
	EnterExecute_statement_arg(c *Execute_statement_argContext)

	// EnterExecute_statement_arg_named is called when entering the execute_statement_arg_named production.
	EnterExecute_statement_arg_named(c *Execute_statement_arg_namedContext)

	// EnterExecute_statement_arg_unnamed is called when entering the execute_statement_arg_unnamed production.
	EnterExecute_statement_arg_unnamed(c *Execute_statement_arg_unnamedContext)

	// EnterExecute_parameter is called when entering the execute_parameter production.
	EnterExecute_parameter(c *Execute_parameterContext)

	// EnterExecute_var_string is called when entering the execute_var_string production.
	EnterExecute_var_string(c *Execute_var_stringContext)

	// EnterSecurity_statement is called when entering the security_statement production.
	EnterSecurity_statement(c *Security_statementContext)

	// EnterPrincipal_id is called when entering the principal_id production.
	EnterPrincipal_id(c *Principal_idContext)

	// EnterCreate_certificate is called when entering the create_certificate production.
	EnterCreate_certificate(c *Create_certificateContext)

	// EnterExisting_keys is called when entering the existing_keys production.
	EnterExisting_keys(c *Existing_keysContext)

	// EnterPrivate_key_options is called when entering the private_key_options production.
	EnterPrivate_key_options(c *Private_key_optionsContext)

	// EnterGenerate_new_keys is called when entering the generate_new_keys production.
	EnterGenerate_new_keys(c *Generate_new_keysContext)

	// EnterDate_options is called when entering the date_options production.
	EnterDate_options(c *Date_optionsContext)

	// EnterOpen_key is called when entering the open_key production.
	EnterOpen_key(c *Open_keyContext)

	// EnterClose_key is called when entering the close_key production.
	EnterClose_key(c *Close_keyContext)

	// EnterCreate_key is called when entering the create_key production.
	EnterCreate_key(c *Create_keyContext)

	// EnterKey_options is called when entering the key_options production.
	EnterKey_options(c *Key_optionsContext)

	// EnterAlgorithm is called when entering the algorithm production.
	EnterAlgorithm(c *AlgorithmContext)

	// EnterEncryption_mechanism is called when entering the encryption_mechanism production.
	EnterEncryption_mechanism(c *Encryption_mechanismContext)

	// EnterDecryption_mechanism is called when entering the decryption_mechanism production.
	EnterDecryption_mechanism(c *Decryption_mechanismContext)

	// EnterGrant_permission is called when entering the grant_permission production.
	EnterGrant_permission(c *Grant_permissionContext)

	// EnterSet_statement is called when entering the set_statement production.
	EnterSet_statement(c *Set_statementContext)

	// EnterTransaction_statement is called when entering the transaction_statement production.
	EnterTransaction_statement(c *Transaction_statementContext)

	// EnterGo_statement is called when entering the go_statement production.
	EnterGo_statement(c *Go_statementContext)

	// EnterUse_statement is called when entering the use_statement production.
	EnterUse_statement(c *Use_statementContext)

	// EnterSetuser_statement is called when entering the setuser_statement production.
	EnterSetuser_statement(c *Setuser_statementContext)

	// EnterReconfigure_statement is called when entering the reconfigure_statement production.
	EnterReconfigure_statement(c *Reconfigure_statementContext)

	// EnterShutdown_statement is called when entering the shutdown_statement production.
	EnterShutdown_statement(c *Shutdown_statementContext)

	// EnterCheckpoint_statement is called when entering the checkpoint_statement production.
	EnterCheckpoint_statement(c *Checkpoint_statementContext)

	// EnterDbcc_checkalloc_option is called when entering the dbcc_checkalloc_option production.
	EnterDbcc_checkalloc_option(c *Dbcc_checkalloc_optionContext)

	// EnterDbcc_checkalloc is called when entering the dbcc_checkalloc production.
	EnterDbcc_checkalloc(c *Dbcc_checkallocContext)

	// EnterDbcc_checkcatalog is called when entering the dbcc_checkcatalog production.
	EnterDbcc_checkcatalog(c *Dbcc_checkcatalogContext)

	// EnterDbcc_checkconstraints_option is called when entering the dbcc_checkconstraints_option production.
	EnterDbcc_checkconstraints_option(c *Dbcc_checkconstraints_optionContext)

	// EnterDbcc_checkconstraints is called when entering the dbcc_checkconstraints production.
	EnterDbcc_checkconstraints(c *Dbcc_checkconstraintsContext)

	// EnterDbcc_checkdb_table_option is called when entering the dbcc_checkdb_table_option production.
	EnterDbcc_checkdb_table_option(c *Dbcc_checkdb_table_optionContext)

	// EnterDbcc_checkdb is called when entering the dbcc_checkdb production.
	EnterDbcc_checkdb(c *Dbcc_checkdbContext)

	// EnterDbcc_checkfilegroup_option is called when entering the dbcc_checkfilegroup_option production.
	EnterDbcc_checkfilegroup_option(c *Dbcc_checkfilegroup_optionContext)

	// EnterDbcc_checkfilegroup is called when entering the dbcc_checkfilegroup production.
	EnterDbcc_checkfilegroup(c *Dbcc_checkfilegroupContext)

	// EnterDbcc_checktable is called when entering the dbcc_checktable production.
	EnterDbcc_checktable(c *Dbcc_checktableContext)

	// EnterDbcc_cleantable is called when entering the dbcc_cleantable production.
	EnterDbcc_cleantable(c *Dbcc_cleantableContext)

	// EnterDbcc_clonedatabase_option is called when entering the dbcc_clonedatabase_option production.
	EnterDbcc_clonedatabase_option(c *Dbcc_clonedatabase_optionContext)

	// EnterDbcc_clonedatabase is called when entering the dbcc_clonedatabase production.
	EnterDbcc_clonedatabase(c *Dbcc_clonedatabaseContext)

	// EnterDbcc_pdw_showspaceused is called when entering the dbcc_pdw_showspaceused production.
	EnterDbcc_pdw_showspaceused(c *Dbcc_pdw_showspaceusedContext)

	// EnterDbcc_proccache is called when entering the dbcc_proccache production.
	EnterDbcc_proccache(c *Dbcc_proccacheContext)

	// EnterDbcc_showcontig_option is called when entering the dbcc_showcontig_option production.
	EnterDbcc_showcontig_option(c *Dbcc_showcontig_optionContext)

	// EnterDbcc_showcontig is called when entering the dbcc_showcontig production.
	EnterDbcc_showcontig(c *Dbcc_showcontigContext)

	// EnterDbcc_shrinklog is called when entering the dbcc_shrinklog production.
	EnterDbcc_shrinklog(c *Dbcc_shrinklogContext)

	// EnterDbcc_dbreindex is called when entering the dbcc_dbreindex production.
	EnterDbcc_dbreindex(c *Dbcc_dbreindexContext)

	// EnterDbcc_dll_free is called when entering the dbcc_dll_free production.
	EnterDbcc_dll_free(c *Dbcc_dll_freeContext)

	// EnterDbcc_dropcleanbuffers is called when entering the dbcc_dropcleanbuffers production.
	EnterDbcc_dropcleanbuffers(c *Dbcc_dropcleanbuffersContext)

	// EnterDbcc_clause is called when entering the dbcc_clause production.
	EnterDbcc_clause(c *Dbcc_clauseContext)

	// EnterExecute_clause is called when entering the execute_clause production.
	EnterExecute_clause(c *Execute_clauseContext)

	// EnterDeclare_local is called when entering the declare_local production.
	EnterDeclare_local(c *Declare_localContext)

	// EnterTable_type_definition is called when entering the table_type_definition production.
	EnterTable_type_definition(c *Table_type_definitionContext)

	// EnterTable_type_indices is called when entering the table_type_indices production.
	EnterTable_type_indices(c *Table_type_indicesContext)

	// EnterXml_type_definition is called when entering the xml_type_definition production.
	EnterXml_type_definition(c *Xml_type_definitionContext)

	// EnterXml_schema_collection is called when entering the xml_schema_collection production.
	EnterXml_schema_collection(c *Xml_schema_collectionContext)

	// EnterColumn_def_table_constraints is called when entering the column_def_table_constraints production.
	EnterColumn_def_table_constraints(c *Column_def_table_constraintsContext)

	// EnterColumn_def_table_constraint is called when entering the column_def_table_constraint production.
	EnterColumn_def_table_constraint(c *Column_def_table_constraintContext)

	// EnterColumn_definition is called when entering the column_definition production.
	EnterColumn_definition(c *Column_definitionContext)

	// EnterColumn_definition_element is called when entering the column_definition_element production.
	EnterColumn_definition_element(c *Column_definition_elementContext)

	// EnterColumn_modifier is called when entering the column_modifier production.
	EnterColumn_modifier(c *Column_modifierContext)

	// EnterMaterialized_column_definition is called when entering the materialized_column_definition production.
	EnterMaterialized_column_definition(c *Materialized_column_definitionContext)

	// EnterColumn_constraint is called when entering the column_constraint production.
	EnterColumn_constraint(c *Column_constraintContext)

	// EnterColumn_index is called when entering the column_index production.
	EnterColumn_index(c *Column_indexContext)

	// EnterOn_partition_or_filegroup is called when entering the on_partition_or_filegroup production.
	EnterOn_partition_or_filegroup(c *On_partition_or_filegroupContext)

	// EnterTable_constraint is called when entering the table_constraint production.
	EnterTable_constraint(c *Table_constraintContext)

	// EnterConnection_node is called when entering the connection_node production.
	EnterConnection_node(c *Connection_nodeContext)

	// EnterPrimary_key_options is called when entering the primary_key_options production.
	EnterPrimary_key_options(c *Primary_key_optionsContext)

	// EnterForeign_key_options is called when entering the foreign_key_options production.
	EnterForeign_key_options(c *Foreign_key_optionsContext)

	// EnterCheck_constraint is called when entering the check_constraint production.
	EnterCheck_constraint(c *Check_constraintContext)

	// EnterOn_delete is called when entering the on_delete production.
	EnterOn_delete(c *On_deleteContext)

	// EnterOn_update is called when entering the on_update production.
	EnterOn_update(c *On_updateContext)

	// EnterAlter_table_index_options is called when entering the alter_table_index_options production.
	EnterAlter_table_index_options(c *Alter_table_index_optionsContext)

	// EnterAlter_table_index_option is called when entering the alter_table_index_option production.
	EnterAlter_table_index_option(c *Alter_table_index_optionContext)

	// EnterDeclare_cursor is called when entering the declare_cursor production.
	EnterDeclare_cursor(c *Declare_cursorContext)

	// EnterDeclare_set_cursor_common is called when entering the declare_set_cursor_common production.
	EnterDeclare_set_cursor_common(c *Declare_set_cursor_commonContext)

	// EnterDeclare_set_cursor_common_partial is called when entering the declare_set_cursor_common_partial production.
	EnterDeclare_set_cursor_common_partial(c *Declare_set_cursor_common_partialContext)

	// EnterFetch_cursor is called when entering the fetch_cursor production.
	EnterFetch_cursor(c *Fetch_cursorContext)

	// EnterSet_special is called when entering the set_special production.
	EnterSet_special(c *Set_specialContext)

	// EnterSpecial_list is called when entering the special_list production.
	EnterSpecial_list(c *Special_listContext)

	// EnterConstant_LOCAL_ID is called when entering the constant_LOCAL_ID production.
	EnterConstant_LOCAL_ID(c *Constant_LOCAL_IDContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterParameter is called when entering the parameter production.
	EnterParameter(c *ParameterContext)

	// EnterTime_zone is called when entering the time_zone production.
	EnterTime_zone(c *Time_zoneContext)

	// EnterPrimitive_expression is called when entering the primitive_expression production.
	EnterPrimitive_expression(c *Primitive_expressionContext)

	// EnterCase_expression is called when entering the case_expression production.
	EnterCase_expression(c *Case_expressionContext)

	// EnterUnary_operator_expression is called when entering the unary_operator_expression production.
	EnterUnary_operator_expression(c *Unary_operator_expressionContext)

	// EnterBracket_expression is called when entering the bracket_expression production.
	EnterBracket_expression(c *Bracket_expressionContext)

	// EnterSubquery is called when entering the subquery production.
	EnterSubquery(c *SubqueryContext)

	// EnterWith_expression is called when entering the with_expression production.
	EnterWith_expression(c *With_expressionContext)

	// EnterCommon_table_expression is called when entering the common_table_expression production.
	EnterCommon_table_expression(c *Common_table_expressionContext)

	// EnterUpdate_elem is called when entering the update_elem production.
	EnterUpdate_elem(c *Update_elemContext)

	// EnterUpdate_elem_merge is called when entering the update_elem_merge production.
	EnterUpdate_elem_merge(c *Update_elem_mergeContext)

	// EnterSearch_condition is called when entering the search_condition production.
	EnterSearch_condition(c *Search_conditionContext)

	// EnterPredicate is called when entering the predicate production.
	EnterPredicate(c *PredicateContext)

	// EnterQuery_expression is called when entering the query_expression production.
	EnterQuery_expression(c *Query_expressionContext)

	// EnterSql_union is called when entering the sql_union production.
	EnterSql_union(c *Sql_unionContext)

	// EnterQuery_specification is called when entering the query_specification production.
	EnterQuery_specification(c *Query_specificationContext)

	// EnterTop_clause is called when entering the top_clause production.
	EnterTop_clause(c *Top_clauseContext)

	// EnterTop_percent is called when entering the top_percent production.
	EnterTop_percent(c *Top_percentContext)

	// EnterTop_count is called when entering the top_count production.
	EnterTop_count(c *Top_countContext)

	// EnterOrder_by_clause is called when entering the order_by_clause production.
	EnterOrder_by_clause(c *Order_by_clauseContext)

	// EnterSelect_order_by_clause is called when entering the select_order_by_clause production.
	EnterSelect_order_by_clause(c *Select_order_by_clauseContext)

	// EnterFor_clause is called when entering the for_clause production.
	EnterFor_clause(c *For_clauseContext)

	// EnterXml_common_directives is called when entering the xml_common_directives production.
	EnterXml_common_directives(c *Xml_common_directivesContext)

	// EnterOrder_by_expression is called when entering the order_by_expression production.
	EnterOrder_by_expression(c *Order_by_expressionContext)

	// EnterGrouping_sets_item is called when entering the grouping_sets_item production.
	EnterGrouping_sets_item(c *Grouping_sets_itemContext)

	// EnterGroup_by_item is called when entering the group_by_item production.
	EnterGroup_by_item(c *Group_by_itemContext)

	// EnterOption_clause is called when entering the option_clause production.
	EnterOption_clause(c *Option_clauseContext)

	// EnterOption is called when entering the option production.
	EnterOption(c *OptionContext)

	// EnterOptimize_for_arg is called when entering the optimize_for_arg production.
	EnterOptimize_for_arg(c *Optimize_for_argContext)

	// EnterSelect_list is called when entering the select_list production.
	EnterSelect_list(c *Select_listContext)

	// EnterUdt_method_arguments is called when entering the udt_method_arguments production.
	EnterUdt_method_arguments(c *Udt_method_argumentsContext)

	// EnterAsterisk is called when entering the asterisk production.
	EnterAsterisk(c *AsteriskContext)

	// EnterUdt_elem is called when entering the udt_elem production.
	EnterUdt_elem(c *Udt_elemContext)

	// EnterExpression_elem is called when entering the expression_elem production.
	EnterExpression_elem(c *Expression_elemContext)

	// EnterSelect_list_elem is called when entering the select_list_elem production.
	EnterSelect_list_elem(c *Select_list_elemContext)

	// EnterTable_sources is called when entering the table_sources production.
	EnterTable_sources(c *Table_sourcesContext)

	// EnterNon_ansi_join is called when entering the non_ansi_join production.
	EnterNon_ansi_join(c *Non_ansi_joinContext)

	// EnterTable_source is called when entering the table_source production.
	EnterTable_source(c *Table_sourceContext)

	// EnterTable_source_item is called when entering the table_source_item production.
	EnterTable_source_item(c *Table_source_itemContext)

	// EnterOpen_xml is called when entering the open_xml production.
	EnterOpen_xml(c *Open_xmlContext)

	// EnterOpen_json is called when entering the open_json production.
	EnterOpen_json(c *Open_jsonContext)

	// EnterJson_declaration is called when entering the json_declaration production.
	EnterJson_declaration(c *Json_declarationContext)

	// EnterJson_column_declaration is called when entering the json_column_declaration production.
	EnterJson_column_declaration(c *Json_column_declarationContext)

	// EnterSchema_declaration is called when entering the schema_declaration production.
	EnterSchema_declaration(c *Schema_declarationContext)

	// EnterColumn_declaration is called when entering the column_declaration production.
	EnterColumn_declaration(c *Column_declarationContext)

	// EnterChange_table is called when entering the change_table production.
	EnterChange_table(c *Change_tableContext)

	// EnterChange_table_changes is called when entering the change_table_changes production.
	EnterChange_table_changes(c *Change_table_changesContext)

	// EnterChange_table_version is called when entering the change_table_version production.
	EnterChange_table_version(c *Change_table_versionContext)

	// EnterJoin_part is called when entering the join_part production.
	EnterJoin_part(c *Join_partContext)

	// EnterJoin_on is called when entering the join_on production.
	EnterJoin_on(c *Join_onContext)

	// EnterCross_join is called when entering the cross_join production.
	EnterCross_join(c *Cross_joinContext)

	// EnterApply_ is called when entering the apply_ production.
	EnterApply_(c *Apply_Context)

	// EnterPivot is called when entering the pivot production.
	EnterPivot(c *PivotContext)

	// EnterUnpivot is called when entering the unpivot production.
	EnterUnpivot(c *UnpivotContext)

	// EnterPivot_clause is called when entering the pivot_clause production.
	EnterPivot_clause(c *Pivot_clauseContext)

	// EnterUnpivot_clause is called when entering the unpivot_clause production.
	EnterUnpivot_clause(c *Unpivot_clauseContext)

	// EnterFull_column_name_list is called when entering the full_column_name_list production.
	EnterFull_column_name_list(c *Full_column_name_listContext)

	// EnterRowset_function is called when entering the rowset_function production.
	EnterRowset_function(c *Rowset_functionContext)

	// EnterBulk_option is called when entering the bulk_option production.
	EnterBulk_option(c *Bulk_optionContext)

	// EnterDerived_table is called when entering the derived_table production.
	EnterDerived_table(c *Derived_tableContext)

	// EnterRANKING_WINDOWED_FUNC is called when entering the RANKING_WINDOWED_FUNC production.
	EnterRANKING_WINDOWED_FUNC(c *RANKING_WINDOWED_FUNCContext)

	// EnterAGGREGATE_WINDOWED_FUNC is called when entering the AGGREGATE_WINDOWED_FUNC production.
	EnterAGGREGATE_WINDOWED_FUNC(c *AGGREGATE_WINDOWED_FUNCContext)

	// EnterANALYTIC_WINDOWED_FUNC is called when entering the ANALYTIC_WINDOWED_FUNC production.
	EnterANALYTIC_WINDOWED_FUNC(c *ANALYTIC_WINDOWED_FUNCContext)

	// EnterBUILT_IN_FUNC is called when entering the BUILT_IN_FUNC production.
	EnterBUILT_IN_FUNC(c *BUILT_IN_FUNCContext)

	// EnterSCALAR_FUNCTION is called when entering the SCALAR_FUNCTION production.
	EnterSCALAR_FUNCTION(c *SCALAR_FUNCTIONContext)

	// EnterFREE_TEXT is called when entering the FREE_TEXT production.
	EnterFREE_TEXT(c *FREE_TEXTContext)

	// EnterPARTITION_FUNC is called when entering the PARTITION_FUNC production.
	EnterPARTITION_FUNC(c *PARTITION_FUNCContext)

	// EnterHIERARCHYID_METHOD is called when entering the HIERARCHYID_METHOD production.
	EnterHIERARCHYID_METHOD(c *HIERARCHYID_METHODContext)

	// EnterPartition_function is called when entering the partition_function production.
	EnterPartition_function(c *Partition_functionContext)

	// EnterFreetext_function is called when entering the freetext_function production.
	EnterFreetext_function(c *Freetext_functionContext)

	// EnterFreetext_predicate is called when entering the freetext_predicate production.
	EnterFreetext_predicate(c *Freetext_predicateContext)

	// EnterJson_key_value is called when entering the json_key_value production.
	EnterJson_key_value(c *Json_key_valueContext)

	// EnterJson_null_clause is called when entering the json_null_clause production.
	EnterJson_null_clause(c *Json_null_clauseContext)

	// EnterAPP_NAME is called when entering the APP_NAME production.
	EnterAPP_NAME(c *APP_NAMEContext)

	// EnterAPPLOCK_MODE is called when entering the APPLOCK_MODE production.
	EnterAPPLOCK_MODE(c *APPLOCK_MODEContext)

	// EnterAPPLOCK_TEST is called when entering the APPLOCK_TEST production.
	EnterAPPLOCK_TEST(c *APPLOCK_TESTContext)

	// EnterASSEMBLYPROPERTY is called when entering the ASSEMBLYPROPERTY production.
	EnterASSEMBLYPROPERTY(c *ASSEMBLYPROPERTYContext)

	// EnterCOL_LENGTH is called when entering the COL_LENGTH production.
	EnterCOL_LENGTH(c *COL_LENGTHContext)

	// EnterCOL_NAME is called when entering the COL_NAME production.
	EnterCOL_NAME(c *COL_NAMEContext)

	// EnterCOLUMNPROPERTY is called when entering the COLUMNPROPERTY production.
	EnterCOLUMNPROPERTY(c *COLUMNPROPERTYContext)

	// EnterDATABASEPROPERTYEX is called when entering the DATABASEPROPERTYEX production.
	EnterDATABASEPROPERTYEX(c *DATABASEPROPERTYEXContext)

	// EnterDB_ID is called when entering the DB_ID production.
	EnterDB_ID(c *DB_IDContext)

	// EnterDB_NAME is called when entering the DB_NAME production.
	EnterDB_NAME(c *DB_NAMEContext)

	// EnterFILE_ID is called when entering the FILE_ID production.
	EnterFILE_ID(c *FILE_IDContext)

	// EnterFILE_IDEX is called when entering the FILE_IDEX production.
	EnterFILE_IDEX(c *FILE_IDEXContext)

	// EnterFILE_NAME is called when entering the FILE_NAME production.
	EnterFILE_NAME(c *FILE_NAMEContext)

	// EnterFILEGROUP_ID is called when entering the FILEGROUP_ID production.
	EnterFILEGROUP_ID(c *FILEGROUP_IDContext)

	// EnterFILEGROUP_NAME is called when entering the FILEGROUP_NAME production.
	EnterFILEGROUP_NAME(c *FILEGROUP_NAMEContext)

	// EnterFILEGROUPPROPERTY is called when entering the FILEGROUPPROPERTY production.
	EnterFILEGROUPPROPERTY(c *FILEGROUPPROPERTYContext)

	// EnterFILEPROPERTY is called when entering the FILEPROPERTY production.
	EnterFILEPROPERTY(c *FILEPROPERTYContext)

	// EnterFILEPROPERTYEX is called when entering the FILEPROPERTYEX production.
	EnterFILEPROPERTYEX(c *FILEPROPERTYEXContext)

	// EnterFULLTEXTCATALOGPROPERTY is called when entering the FULLTEXTCATALOGPROPERTY production.
	EnterFULLTEXTCATALOGPROPERTY(c *FULLTEXTCATALOGPROPERTYContext)

	// EnterFULLTEXTSERVICEPROPERTY is called when entering the FULLTEXTSERVICEPROPERTY production.
	EnterFULLTEXTSERVICEPROPERTY(c *FULLTEXTSERVICEPROPERTYContext)

	// EnterINDEX_COL is called when entering the INDEX_COL production.
	EnterINDEX_COL(c *INDEX_COLContext)

	// EnterINDEXKEY_PROPERTY is called when entering the INDEXKEY_PROPERTY production.
	EnterINDEXKEY_PROPERTY(c *INDEXKEY_PROPERTYContext)

	// EnterINDEXPROPERTY is called when entering the INDEXPROPERTY production.
	EnterINDEXPROPERTY(c *INDEXPROPERTYContext)

	// EnterNEXT_VALUE_FOR is called when entering the NEXT_VALUE_FOR production.
	EnterNEXT_VALUE_FOR(c *NEXT_VALUE_FORContext)

	// EnterOBJECT_DEFINITION is called when entering the OBJECT_DEFINITION production.
	EnterOBJECT_DEFINITION(c *OBJECT_DEFINITIONContext)

	// EnterOBJECT_ID is called when entering the OBJECT_ID production.
	EnterOBJECT_ID(c *OBJECT_IDContext)

	// EnterOBJECT_NAME is called when entering the OBJECT_NAME production.
	EnterOBJECT_NAME(c *OBJECT_NAMEContext)

	// EnterOBJECT_SCHEMA_NAME is called when entering the OBJECT_SCHEMA_NAME production.
	EnterOBJECT_SCHEMA_NAME(c *OBJECT_SCHEMA_NAMEContext)

	// EnterOBJECTPROPERTY is called when entering the OBJECTPROPERTY production.
	EnterOBJECTPROPERTY(c *OBJECTPROPERTYContext)

	// EnterOBJECTPROPERTYEX is called when entering the OBJECTPROPERTYEX production.
	EnterOBJECTPROPERTYEX(c *OBJECTPROPERTYEXContext)

	// EnterORIGINAL_DB_NAME is called when entering the ORIGINAL_DB_NAME production.
	EnterORIGINAL_DB_NAME(c *ORIGINAL_DB_NAMEContext)

	// EnterPARSENAME is called when entering the PARSENAME production.
	EnterPARSENAME(c *PARSENAMEContext)

	// EnterSCHEMA_ID is called when entering the SCHEMA_ID production.
	EnterSCHEMA_ID(c *SCHEMA_IDContext)

	// EnterSCHEMA_NAME is called when entering the SCHEMA_NAME production.
	EnterSCHEMA_NAME(c *SCHEMA_NAMEContext)

	// EnterSCOPE_IDENTITY is called when entering the SCOPE_IDENTITY production.
	EnterSCOPE_IDENTITY(c *SCOPE_IDENTITYContext)

	// EnterSERVERPROPERTY is called when entering the SERVERPROPERTY production.
	EnterSERVERPROPERTY(c *SERVERPROPERTYContext)

	// EnterSTATS_DATE is called when entering the STATS_DATE production.
	EnterSTATS_DATE(c *STATS_DATEContext)

	// EnterTYPE_ID is called when entering the TYPE_ID production.
	EnterTYPE_ID(c *TYPE_IDContext)

	// EnterTYPE_NAME is called when entering the TYPE_NAME production.
	EnterTYPE_NAME(c *TYPE_NAMEContext)

	// EnterTYPEPROPERTY is called when entering the TYPEPROPERTY production.
	EnterTYPEPROPERTY(c *TYPEPROPERTYContext)

	// EnterASCII is called when entering the ASCII production.
	EnterASCII(c *ASCIIContext)

	// EnterCHAR is called when entering the CHAR production.
	EnterCHAR(c *CHARContext)

	// EnterCHARINDEX is called when entering the CHARINDEX production.
	EnterCHARINDEX(c *CHARINDEXContext)

	// EnterCONCAT is called when entering the CONCAT production.
	EnterCONCAT(c *CONCATContext)

	// EnterCONCAT_WS is called when entering the CONCAT_WS production.
	EnterCONCAT_WS(c *CONCAT_WSContext)

	// EnterDIFFERENCE is called when entering the DIFFERENCE production.
	EnterDIFFERENCE(c *DIFFERENCEContext)

	// EnterFORMAT is called when entering the FORMAT production.
	EnterFORMAT(c *FORMATContext)

	// EnterLEFT is called when entering the LEFT production.
	EnterLEFT(c *LEFTContext)

	// EnterLEN is called when entering the LEN production.
	EnterLEN(c *LENContext)

	// EnterLOWER is called when entering the LOWER production.
	EnterLOWER(c *LOWERContext)

	// EnterLTRIM is called when entering the LTRIM production.
	EnterLTRIM(c *LTRIMContext)

	// EnterNCHAR is called when entering the NCHAR production.
	EnterNCHAR(c *NCHARContext)

	// EnterPATINDEX is called when entering the PATINDEX production.
	EnterPATINDEX(c *PATINDEXContext)

	// EnterQUOTENAME is called when entering the QUOTENAME production.
	EnterQUOTENAME(c *QUOTENAMEContext)

	// EnterREPLACE is called when entering the REPLACE production.
	EnterREPLACE(c *REPLACEContext)

	// EnterREPLICATE is called when entering the REPLICATE production.
	EnterREPLICATE(c *REPLICATEContext)

	// EnterREVERSE is called when entering the REVERSE production.
	EnterREVERSE(c *REVERSEContext)

	// EnterRIGHT is called when entering the RIGHT production.
	EnterRIGHT(c *RIGHTContext)

	// EnterRTRIM is called when entering the RTRIM production.
	EnterRTRIM(c *RTRIMContext)

	// EnterSOUNDEX is called when entering the SOUNDEX production.
	EnterSOUNDEX(c *SOUNDEXContext)

	// EnterSPACE is called when entering the SPACE production.
	EnterSPACE(c *SPACEContext)

	// EnterSTR is called when entering the STR production.
	EnterSTR(c *STRContext)

	// EnterSTRINGAGG is called when entering the STRINGAGG production.
	EnterSTRINGAGG(c *STRINGAGGContext)

	// EnterSTRING_ESCAPE is called when entering the STRING_ESCAPE production.
	EnterSTRING_ESCAPE(c *STRING_ESCAPEContext)

	// EnterSTUFF is called when entering the STUFF production.
	EnterSTUFF(c *STUFFContext)

	// EnterSUBSTRING is called when entering the SUBSTRING production.
	EnterSUBSTRING(c *SUBSTRINGContext)

	// EnterTRANSLATE is called when entering the TRANSLATE production.
	EnterTRANSLATE(c *TRANSLATEContext)

	// EnterTRIM is called when entering the TRIM production.
	EnterTRIM(c *TRIMContext)

	// EnterUNICODE is called when entering the UNICODE production.
	EnterUNICODE(c *UNICODEContext)

	// EnterUPPER is called when entering the UPPER production.
	EnterUPPER(c *UPPERContext)

	// EnterBINARY_CHECKSUM is called when entering the BINARY_CHECKSUM production.
	EnterBINARY_CHECKSUM(c *BINARY_CHECKSUMContext)

	// EnterCHECKSUM is called when entering the CHECKSUM production.
	EnterCHECKSUM(c *CHECKSUMContext)

	// EnterCOMPRESS is called when entering the COMPRESS production.
	EnterCOMPRESS(c *COMPRESSContext)

	// EnterCONNECTIONPROPERTY is called when entering the CONNECTIONPROPERTY production.
	EnterCONNECTIONPROPERTY(c *CONNECTIONPROPERTYContext)

	// EnterCONTEXT_INFO is called when entering the CONTEXT_INFO production.
	EnterCONTEXT_INFO(c *CONTEXT_INFOContext)

	// EnterCURRENT_REQUEST_ID is called when entering the CURRENT_REQUEST_ID production.
	EnterCURRENT_REQUEST_ID(c *CURRENT_REQUEST_IDContext)

	// EnterCURRENT_TRANSACTION_ID is called when entering the CURRENT_TRANSACTION_ID production.
	EnterCURRENT_TRANSACTION_ID(c *CURRENT_TRANSACTION_IDContext)

	// EnterDECOMPRESS is called when entering the DECOMPRESS production.
	EnterDECOMPRESS(c *DECOMPRESSContext)

	// EnterERROR_LINE is called when entering the ERROR_LINE production.
	EnterERROR_LINE(c *ERROR_LINEContext)

	// EnterERROR_MESSAGE is called when entering the ERROR_MESSAGE production.
	EnterERROR_MESSAGE(c *ERROR_MESSAGEContext)

	// EnterERROR_NUMBER is called when entering the ERROR_NUMBER production.
	EnterERROR_NUMBER(c *ERROR_NUMBERContext)

	// EnterERROR_PROCEDURE is called when entering the ERROR_PROCEDURE production.
	EnterERROR_PROCEDURE(c *ERROR_PROCEDUREContext)

	// EnterERROR_SEVERITY is called when entering the ERROR_SEVERITY production.
	EnterERROR_SEVERITY(c *ERROR_SEVERITYContext)

	// EnterERROR_STATE is called when entering the ERROR_STATE production.
	EnterERROR_STATE(c *ERROR_STATEContext)

	// EnterFORMATMESSAGE is called when entering the FORMATMESSAGE production.
	EnterFORMATMESSAGE(c *FORMATMESSAGEContext)

	// EnterGET_FILESTREAM_TRANSACTION_CONTEXT is called when entering the GET_FILESTREAM_TRANSACTION_CONTEXT production.
	EnterGET_FILESTREAM_TRANSACTION_CONTEXT(c *GET_FILESTREAM_TRANSACTION_CONTEXTContext)

	// EnterGETANSINULL is called when entering the GETANSINULL production.
	EnterGETANSINULL(c *GETANSINULLContext)

	// EnterHOST_ID is called when entering the HOST_ID production.
	EnterHOST_ID(c *HOST_IDContext)

	// EnterHOST_NAME is called when entering the HOST_NAME production.
	EnterHOST_NAME(c *HOST_NAMEContext)

	// EnterISNULL is called when entering the ISNULL production.
	EnterISNULL(c *ISNULLContext)

	// EnterISNUMERIC is called when entering the ISNUMERIC production.
	EnterISNUMERIC(c *ISNUMERICContext)

	// EnterMIN_ACTIVE_ROWVERSION is called when entering the MIN_ACTIVE_ROWVERSION production.
	EnterMIN_ACTIVE_ROWVERSION(c *MIN_ACTIVE_ROWVERSIONContext)

	// EnterNEWID is called when entering the NEWID production.
	EnterNEWID(c *NEWIDContext)

	// EnterNEWSEQUENTIALID is called when entering the NEWSEQUENTIALID production.
	EnterNEWSEQUENTIALID(c *NEWSEQUENTIALIDContext)

	// EnterROWCOUNT_BIG is called when entering the ROWCOUNT_BIG production.
	EnterROWCOUNT_BIG(c *ROWCOUNT_BIGContext)

	// EnterSESSION_CONTEXT is called when entering the SESSION_CONTEXT production.
	EnterSESSION_CONTEXT(c *SESSION_CONTEXTContext)

	// EnterXACT_STATE is called when entering the XACT_STATE production.
	EnterXACT_STATE(c *XACT_STATEContext)

	// EnterCAST is called when entering the CAST production.
	EnterCAST(c *CASTContext)

	// EnterTRY_CAST is called when entering the TRY_CAST production.
	EnterTRY_CAST(c *TRY_CASTContext)

	// EnterCONVERT is called when entering the CONVERT production.
	EnterCONVERT(c *CONVERTContext)

	// EnterCOALESCE is called when entering the COALESCE production.
	EnterCOALESCE(c *COALESCEContext)

	// EnterCURSOR_ROWS is called when entering the CURSOR_ROWS production.
	EnterCURSOR_ROWS(c *CURSOR_ROWSContext)

	// EnterFETCH_STATUS is called when entering the FETCH_STATUS production.
	EnterFETCH_STATUS(c *FETCH_STATUSContext)

	// EnterCURSOR_STATUS is called when entering the CURSOR_STATUS production.
	EnterCURSOR_STATUS(c *CURSOR_STATUSContext)

	// EnterCERT_ID is called when entering the CERT_ID production.
	EnterCERT_ID(c *CERT_IDContext)

	// EnterDATALENGTH is called when entering the DATALENGTH production.
	EnterDATALENGTH(c *DATALENGTHContext)

	// EnterIDENT_CURRENT is called when entering the IDENT_CURRENT production.
	EnterIDENT_CURRENT(c *IDENT_CURRENTContext)

	// EnterIDENT_INCR is called when entering the IDENT_INCR production.
	EnterIDENT_INCR(c *IDENT_INCRContext)

	// EnterIDENT_SEED is called when entering the IDENT_SEED production.
	EnterIDENT_SEED(c *IDENT_SEEDContext)

	// EnterIDENTITY is called when entering the IDENTITY production.
	EnterIDENTITY(c *IDENTITYContext)

	// EnterSQL_VARIANT_PROPERTY is called when entering the SQL_VARIANT_PROPERTY production.
	EnterSQL_VARIANT_PROPERTY(c *SQL_VARIANT_PROPERTYContext)

	// EnterCURRENT_DATE is called when entering the CURRENT_DATE production.
	EnterCURRENT_DATE(c *CURRENT_DATEContext)

	// EnterCURRENT_TIMESTAMP is called when entering the CURRENT_TIMESTAMP production.
	EnterCURRENT_TIMESTAMP(c *CURRENT_TIMESTAMPContext)

	// EnterCURRENT_TIMEZONE is called when entering the CURRENT_TIMEZONE production.
	EnterCURRENT_TIMEZONE(c *CURRENT_TIMEZONEContext)

	// EnterCURRENT_TIMEZONE_ID is called when entering the CURRENT_TIMEZONE_ID production.
	EnterCURRENT_TIMEZONE_ID(c *CURRENT_TIMEZONE_IDContext)

	// EnterDATE_BUCKET is called when entering the DATE_BUCKET production.
	EnterDATE_BUCKET(c *DATE_BUCKETContext)

	// EnterDATEADD is called when entering the DATEADD production.
	EnterDATEADD(c *DATEADDContext)

	// EnterDATEDIFF is called when entering the DATEDIFF production.
	EnterDATEDIFF(c *DATEDIFFContext)

	// EnterDATEDIFF_BIG is called when entering the DATEDIFF_BIG production.
	EnterDATEDIFF_BIG(c *DATEDIFF_BIGContext)

	// EnterDATEFROMPARTS is called when entering the DATEFROMPARTS production.
	EnterDATEFROMPARTS(c *DATEFROMPARTSContext)

	// EnterDATENAME is called when entering the DATENAME production.
	EnterDATENAME(c *DATENAMEContext)

	// EnterDATEPART is called when entering the DATEPART production.
	EnterDATEPART(c *DATEPARTContext)

	// EnterDATETIME2FROMPARTS is called when entering the DATETIME2FROMPARTS production.
	EnterDATETIME2FROMPARTS(c *DATETIME2FROMPARTSContext)

	// EnterDATETIMEFROMPARTS is called when entering the DATETIMEFROMPARTS production.
	EnterDATETIMEFROMPARTS(c *DATETIMEFROMPARTSContext)

	// EnterDATETIMEOFFSETFROMPARTS is called when entering the DATETIMEOFFSETFROMPARTS production.
	EnterDATETIMEOFFSETFROMPARTS(c *DATETIMEOFFSETFROMPARTSContext)

	// EnterDATETRUNC is called when entering the DATETRUNC production.
	EnterDATETRUNC(c *DATETRUNCContext)

	// EnterDAY is called when entering the DAY production.
	EnterDAY(c *DAYContext)

	// EnterEOMONTH is called when entering the EOMONTH production.
	EnterEOMONTH(c *EOMONTHContext)

	// EnterGETDATE is called when entering the GETDATE production.
	EnterGETDATE(c *GETDATEContext)

	// EnterGETUTCDATE is called when entering the GETUTCDATE production.
	EnterGETUTCDATE(c *GETUTCDATEContext)

	// EnterISDATE is called when entering the ISDATE production.
	EnterISDATE(c *ISDATEContext)

	// EnterMONTH is called when entering the MONTH production.
	EnterMONTH(c *MONTHContext)

	// EnterSMALLDATETIMEFROMPARTS is called when entering the SMALLDATETIMEFROMPARTS production.
	EnterSMALLDATETIMEFROMPARTS(c *SMALLDATETIMEFROMPARTSContext)

	// EnterSWITCHOFFSET is called when entering the SWITCHOFFSET production.
	EnterSWITCHOFFSET(c *SWITCHOFFSETContext)

	// EnterSYSDATETIME is called when entering the SYSDATETIME production.
	EnterSYSDATETIME(c *SYSDATETIMEContext)

	// EnterSYSDATETIMEOFFSET is called when entering the SYSDATETIMEOFFSET production.
	EnterSYSDATETIMEOFFSET(c *SYSDATETIMEOFFSETContext)

	// EnterSYSUTCDATETIME is called when entering the SYSUTCDATETIME production.
	EnterSYSUTCDATETIME(c *SYSUTCDATETIMEContext)

	// EnterTIMEFROMPARTS is called when entering the TIMEFROMPARTS production.
	EnterTIMEFROMPARTS(c *TIMEFROMPARTSContext)

	// EnterTODATETIMEOFFSET is called when entering the TODATETIMEOFFSET production.
	EnterTODATETIMEOFFSET(c *TODATETIMEOFFSETContext)

	// EnterYEAR is called when entering the YEAR production.
	EnterYEAR(c *YEARContext)

	// EnterNULLIF is called when entering the NULLIF production.
	EnterNULLIF(c *NULLIFContext)

	// EnterPARSE is called when entering the PARSE production.
	EnterPARSE(c *PARSEContext)

	// EnterXML_DATA_TYPE_FUNC is called when entering the XML_DATA_TYPE_FUNC production.
	EnterXML_DATA_TYPE_FUNC(c *XML_DATA_TYPE_FUNCContext)

	// EnterIIF is called when entering the IIF production.
	EnterIIF(c *IIFContext)

	// EnterISJSON is called when entering the ISJSON production.
	EnterISJSON(c *ISJSONContext)

	// EnterJSON_OBJECT is called when entering the JSON_OBJECT production.
	EnterJSON_OBJECT(c *JSON_OBJECTContext)

	// EnterJSON_ARRAY is called when entering the JSON_ARRAY production.
	EnterJSON_ARRAY(c *JSON_ARRAYContext)

	// EnterJSON_VALUE is called when entering the JSON_VALUE production.
	EnterJSON_VALUE(c *JSON_VALUEContext)

	// EnterJSON_QUERY is called when entering the JSON_QUERY production.
	EnterJSON_QUERY(c *JSON_QUERYContext)

	// EnterJSON_MODIFY is called when entering the JSON_MODIFY production.
	EnterJSON_MODIFY(c *JSON_MODIFYContext)

	// EnterJSON_PATH_EXISTS is called when entering the JSON_PATH_EXISTS production.
	EnterJSON_PATH_EXISTS(c *JSON_PATH_EXISTSContext)

	// EnterABS is called when entering the ABS production.
	EnterABS(c *ABSContext)

	// EnterACOS is called when entering the ACOS production.
	EnterACOS(c *ACOSContext)

	// EnterASIN is called when entering the ASIN production.
	EnterASIN(c *ASINContext)

	// EnterATAN is called when entering the ATAN production.
	EnterATAN(c *ATANContext)

	// EnterATN2 is called when entering the ATN2 production.
	EnterATN2(c *ATN2Context)

	// EnterCEILING is called when entering the CEILING production.
	EnterCEILING(c *CEILINGContext)

	// EnterCOS is called when entering the COS production.
	EnterCOS(c *COSContext)

	// EnterCOT is called when entering the COT production.
	EnterCOT(c *COTContext)

	// EnterDEGREES is called when entering the DEGREES production.
	EnterDEGREES(c *DEGREESContext)

	// EnterEXP is called when entering the EXP production.
	EnterEXP(c *EXPContext)

	// EnterFLOOR is called when entering the FLOOR production.
	EnterFLOOR(c *FLOORContext)

	// EnterLOG is called when entering the LOG production.
	EnterLOG(c *LOGContext)

	// EnterLOG10 is called when entering the LOG10 production.
	EnterLOG10(c *LOG10Context)

	// EnterPI is called when entering the PI production.
	EnterPI(c *PIContext)

	// EnterPOWER is called when entering the POWER production.
	EnterPOWER(c *POWERContext)

	// EnterRADIANS is called when entering the RADIANS production.
	EnterRADIANS(c *RADIANSContext)

	// EnterRAND is called when entering the RAND production.
	EnterRAND(c *RANDContext)

	// EnterROUND is called when entering the ROUND production.
	EnterROUND(c *ROUNDContext)

	// EnterMATH_SIGN is called when entering the MATH_SIGN production.
	EnterMATH_SIGN(c *MATH_SIGNContext)

	// EnterSIN is called when entering the SIN production.
	EnterSIN(c *SINContext)

	// EnterSQRT is called when entering the SQRT production.
	EnterSQRT(c *SQRTContext)

	// EnterSQUARE is called when entering the SQUARE production.
	EnterSQUARE(c *SQUAREContext)

	// EnterTAN is called when entering the TAN production.
	EnterTAN(c *TANContext)

	// EnterGREATEST is called when entering the GREATEST production.
	EnterGREATEST(c *GREATESTContext)

	// EnterLEAST is called when entering the LEAST production.
	EnterLEAST(c *LEASTContext)

	// EnterCERTENCODED is called when entering the CERTENCODED production.
	EnterCERTENCODED(c *CERTENCODEDContext)

	// EnterCERTPRIVATEKEY is called when entering the CERTPRIVATEKEY production.
	EnterCERTPRIVATEKEY(c *CERTPRIVATEKEYContext)

	// EnterCURRENT_USER is called when entering the CURRENT_USER production.
	EnterCURRENT_USER(c *CURRENT_USERContext)

	// EnterDATABASE_PRINCIPAL_ID is called when entering the DATABASE_PRINCIPAL_ID production.
	EnterDATABASE_PRINCIPAL_ID(c *DATABASE_PRINCIPAL_IDContext)

	// EnterHAS_DBACCESS is called when entering the HAS_DBACCESS production.
	EnterHAS_DBACCESS(c *HAS_DBACCESSContext)

	// EnterHAS_PERMS_BY_NAME is called when entering the HAS_PERMS_BY_NAME production.
	EnterHAS_PERMS_BY_NAME(c *HAS_PERMS_BY_NAMEContext)

	// EnterIS_MEMBER is called when entering the IS_MEMBER production.
	EnterIS_MEMBER(c *IS_MEMBERContext)

	// EnterIS_ROLEMEMBER is called when entering the IS_ROLEMEMBER production.
	EnterIS_ROLEMEMBER(c *IS_ROLEMEMBERContext)

	// EnterIS_SRVROLEMEMBER is called when entering the IS_SRVROLEMEMBER production.
	EnterIS_SRVROLEMEMBER(c *IS_SRVROLEMEMBERContext)

	// EnterLOGINPROPERTY is called when entering the LOGINPROPERTY production.
	EnterLOGINPROPERTY(c *LOGINPROPERTYContext)

	// EnterORIGINAL_LOGIN is called when entering the ORIGINAL_LOGIN production.
	EnterORIGINAL_LOGIN(c *ORIGINAL_LOGINContext)

	// EnterPERMISSIONS is called when entering the PERMISSIONS production.
	EnterPERMISSIONS(c *PERMISSIONSContext)

	// EnterPWDENCRYPT is called when entering the PWDENCRYPT production.
	EnterPWDENCRYPT(c *PWDENCRYPTContext)

	// EnterPWDCOMPARE is called when entering the PWDCOMPARE production.
	EnterPWDCOMPARE(c *PWDCOMPAREContext)

	// EnterSESSION_USER is called when entering the SESSION_USER production.
	EnterSESSION_USER(c *SESSION_USERContext)

	// EnterSESSIONPROPERTY is called when entering the SESSIONPROPERTY production.
	EnterSESSIONPROPERTY(c *SESSIONPROPERTYContext)

	// EnterSUSER_ID is called when entering the SUSER_ID production.
	EnterSUSER_ID(c *SUSER_IDContext)

	// EnterSUSER_SNAME is called when entering the SUSER_SNAME production.
	EnterSUSER_SNAME(c *SUSER_SNAMEContext)

	// EnterSUSER_SID is called when entering the SUSER_SID production.
	EnterSUSER_SID(c *SUSER_SIDContext)

	// EnterSYSTEM_USER is called when entering the SYSTEM_USER production.
	EnterSYSTEM_USER(c *SYSTEM_USERContext)

	// EnterUSER is called when entering the USER production.
	EnterUSER(c *USERContext)

	// EnterUSER_ID is called when entering the USER_ID production.
	EnterUSER_ID(c *USER_IDContext)

	// EnterUSER_NAME is called when entering the USER_NAME production.
	EnterUSER_NAME(c *USER_NAMEContext)

	// EnterXml_data_type_methods is called when entering the xml_data_type_methods production.
	EnterXml_data_type_methods(c *Xml_data_type_methodsContext)

	// EnterDateparts_9 is called when entering the dateparts_9 production.
	EnterDateparts_9(c *Dateparts_9Context)

	// EnterDateparts_12 is called when entering the dateparts_12 production.
	EnterDateparts_12(c *Dateparts_12Context)

	// EnterDateparts_15 is called when entering the dateparts_15 production.
	EnterDateparts_15(c *Dateparts_15Context)

	// EnterDateparts_datetrunc is called when entering the dateparts_datetrunc production.
	EnterDateparts_datetrunc(c *Dateparts_datetruncContext)

	// EnterValue_method is called when entering the value_method production.
	EnterValue_method(c *Value_methodContext)

	// EnterValue_call is called when entering the value_call production.
	EnterValue_call(c *Value_callContext)

	// EnterQuery_method is called when entering the query_method production.
	EnterQuery_method(c *Query_methodContext)

	// EnterQuery_call is called when entering the query_call production.
	EnterQuery_call(c *Query_callContext)

	// EnterExist_method is called when entering the exist_method production.
	EnterExist_method(c *Exist_methodContext)

	// EnterExist_call is called when entering the exist_call production.
	EnterExist_call(c *Exist_callContext)

	// EnterModify_method is called when entering the modify_method production.
	EnterModify_method(c *Modify_methodContext)

	// EnterModify_call is called when entering the modify_call production.
	EnterModify_call(c *Modify_callContext)

	// EnterHierarchyid_call is called when entering the hierarchyid_call production.
	EnterHierarchyid_call(c *Hierarchyid_callContext)

	// EnterHierarchyid_static_method is called when entering the hierarchyid_static_method production.
	EnterHierarchyid_static_method(c *Hierarchyid_static_methodContext)

	// EnterNodes_method is called when entering the nodes_method production.
	EnterNodes_method(c *Nodes_methodContext)

	// EnterSwitch_section is called when entering the switch_section production.
	EnterSwitch_section(c *Switch_sectionContext)

	// EnterSwitch_search_condition_section is called when entering the switch_search_condition_section production.
	EnterSwitch_search_condition_section(c *Switch_search_condition_sectionContext)

	// EnterAs_column_alias is called when entering the as_column_alias production.
	EnterAs_column_alias(c *As_column_aliasContext)

	// EnterAs_table_alias is called when entering the as_table_alias production.
	EnterAs_table_alias(c *As_table_aliasContext)

	// EnterTable_alias is called when entering the table_alias production.
	EnterTable_alias(c *Table_aliasContext)

	// EnterWith_table_hints is called when entering the with_table_hints production.
	EnterWith_table_hints(c *With_table_hintsContext)

	// EnterDeprecated_table_hint is called when entering the deprecated_table_hint production.
	EnterDeprecated_table_hint(c *Deprecated_table_hintContext)

	// EnterSybase_legacy_hints is called when entering the sybase_legacy_hints production.
	EnterSybase_legacy_hints(c *Sybase_legacy_hintsContext)

	// EnterSybase_legacy_hint is called when entering the sybase_legacy_hint production.
	EnterSybase_legacy_hint(c *Sybase_legacy_hintContext)

	// EnterTable_hint is called when entering the table_hint production.
	EnterTable_hint(c *Table_hintContext)

	// EnterIndex_value is called when entering the index_value production.
	EnterIndex_value(c *Index_valueContext)

	// EnterColumn_alias_list is called when entering the column_alias_list production.
	EnterColumn_alias_list(c *Column_alias_listContext)

	// EnterColumn_alias is called when entering the column_alias production.
	EnterColumn_alias(c *Column_aliasContext)

	// EnterTable_value_constructor is called when entering the table_value_constructor production.
	EnterTable_value_constructor(c *Table_value_constructorContext)

	// EnterExpression_list_ is called when entering the expression_list_ production.
	EnterExpression_list_(c *Expression_list_Context)

	// EnterRanking_windowed_function is called when entering the ranking_windowed_function production.
	EnterRanking_windowed_function(c *Ranking_windowed_functionContext)

	// EnterAggregate_windowed_function is called when entering the aggregate_windowed_function production.
	EnterAggregate_windowed_function(c *Aggregate_windowed_functionContext)

	// EnterAnalytic_windowed_function is called when entering the analytic_windowed_function production.
	EnterAnalytic_windowed_function(c *Analytic_windowed_functionContext)

	// EnterAll_distinct_expression is called when entering the all_distinct_expression production.
	EnterAll_distinct_expression(c *All_distinct_expressionContext)

	// EnterOver_clause is called when entering the over_clause production.
	EnterOver_clause(c *Over_clauseContext)

	// EnterRow_or_range_clause is called when entering the row_or_range_clause production.
	EnterRow_or_range_clause(c *Row_or_range_clauseContext)

	// EnterWindow_frame_extent is called when entering the window_frame_extent production.
	EnterWindow_frame_extent(c *Window_frame_extentContext)

	// EnterWindow_frame_bound is called when entering the window_frame_bound production.
	EnterWindow_frame_bound(c *Window_frame_boundContext)

	// EnterWindow_frame_preceding is called when entering the window_frame_preceding production.
	EnterWindow_frame_preceding(c *Window_frame_precedingContext)

	// EnterWindow_frame_following is called when entering the window_frame_following production.
	EnterWindow_frame_following(c *Window_frame_followingContext)

	// EnterCreate_database_option is called when entering the create_database_option production.
	EnterCreate_database_option(c *Create_database_optionContext)

	// EnterDatabase_filestream_option is called when entering the database_filestream_option production.
	EnterDatabase_filestream_option(c *Database_filestream_optionContext)

	// EnterDatabase_file_spec is called when entering the database_file_spec production.
	EnterDatabase_file_spec(c *Database_file_specContext)

	// EnterFile_group is called when entering the file_group production.
	EnterFile_group(c *File_groupContext)

	// EnterFile_spec is called when entering the file_spec production.
	EnterFile_spec(c *File_specContext)

	// EnterEntity_name is called when entering the entity_name production.
	EnterEntity_name(c *Entity_nameContext)

	// EnterEntity_name_for_azure_dw is called when entering the entity_name_for_azure_dw production.
	EnterEntity_name_for_azure_dw(c *Entity_name_for_azure_dwContext)

	// EnterEntity_name_for_parallel_dw is called when entering the entity_name_for_parallel_dw production.
	EnterEntity_name_for_parallel_dw(c *Entity_name_for_parallel_dwContext)

	// EnterFull_table_name is called when entering the full_table_name production.
	EnterFull_table_name(c *Full_table_nameContext)

	// EnterTable_name is called when entering the table_name production.
	EnterTable_name(c *Table_nameContext)

	// EnterSimple_name is called when entering the simple_name production.
	EnterSimple_name(c *Simple_nameContext)

	// EnterFunc_proc_name_schema is called when entering the func_proc_name_schema production.
	EnterFunc_proc_name_schema(c *Func_proc_name_schemaContext)

	// EnterFunc_proc_name_database_schema is called when entering the func_proc_name_database_schema production.
	EnterFunc_proc_name_database_schema(c *Func_proc_name_database_schemaContext)

	// EnterFunc_proc_name_server_database_schema is called when entering the func_proc_name_server_database_schema production.
	EnterFunc_proc_name_server_database_schema(c *Func_proc_name_server_database_schemaContext)

	// EnterDdl_object is called when entering the ddl_object production.
	EnterDdl_object(c *Ddl_objectContext)

	// EnterFull_column_name is called when entering the full_column_name production.
	EnterFull_column_name(c *Full_column_nameContext)

	// EnterColumn_name_list_with_order is called when entering the column_name_list_with_order production.
	EnterColumn_name_list_with_order(c *Column_name_list_with_orderContext)

	// EnterInsert_column_name_list is called when entering the insert_column_name_list production.
	EnterInsert_column_name_list(c *Insert_column_name_listContext)

	// EnterInsert_column_id is called when entering the insert_column_id production.
	EnterInsert_column_id(c *Insert_column_idContext)

	// EnterColumn_name_list is called when entering the column_name_list production.
	EnterColumn_name_list(c *Column_name_listContext)

	// EnterCursor_name is called when entering the cursor_name production.
	EnterCursor_name(c *Cursor_nameContext)

	// EnterOn_off is called when entering the on_off production.
	EnterOn_off(c *On_offContext)

	// EnterClustered is called when entering the clustered production.
	EnterClustered(c *ClusteredContext)

	// EnterNull_notnull is called when entering the null_notnull production.
	EnterNull_notnull(c *Null_notnullContext)

	// EnterScalar_function_name is called when entering the scalar_function_name production.
	EnterScalar_function_name(c *Scalar_function_nameContext)

	// EnterBegin_conversation_timer is called when entering the begin_conversation_timer production.
	EnterBegin_conversation_timer(c *Begin_conversation_timerContext)

	// EnterBegin_conversation_dialog is called when entering the begin_conversation_dialog production.
	EnterBegin_conversation_dialog(c *Begin_conversation_dialogContext)

	// EnterContract_name is called when entering the contract_name production.
	EnterContract_name(c *Contract_nameContext)

	// EnterService_name is called when entering the service_name production.
	EnterService_name(c *Service_nameContext)

	// EnterEnd_conversation is called when entering the end_conversation production.
	EnterEnd_conversation(c *End_conversationContext)

	// EnterWaitfor_conversation is called when entering the waitfor_conversation production.
	EnterWaitfor_conversation(c *Waitfor_conversationContext)

	// EnterGet_conversation is called when entering the get_conversation production.
	EnterGet_conversation(c *Get_conversationContext)

	// EnterQueue_id is called when entering the queue_id production.
	EnterQueue_id(c *Queue_idContext)

	// EnterSend_conversation is called when entering the send_conversation production.
	EnterSend_conversation(c *Send_conversationContext)

	// EnterData_type is called when entering the data_type production.
	EnterData_type(c *Data_typeContext)

	// EnterConstant is called when entering the constant production.
	EnterConstant(c *ConstantContext)

	// EnterPrimitive_constant is called when entering the primitive_constant production.
	EnterPrimitive_constant(c *Primitive_constantContext)

	// EnterKeyword is called when entering the keyword production.
	EnterKeyword(c *KeywordContext)

	// EnterId_ is called when entering the id_ production.
	EnterId_(c *Id_Context)

	// EnterSimple_id is called when entering the simple_id production.
	EnterSimple_id(c *Simple_idContext)

	// EnterId_or_string is called when entering the id_or_string production.
	EnterId_or_string(c *Id_or_stringContext)

	// EnterComparison_operator is called when entering the comparison_operator production.
	EnterComparison_operator(c *Comparison_operatorContext)

	// EnterAssignment_operator is called when entering the assignment_operator production.
	EnterAssignment_operator(c *Assignment_operatorContext)

	// EnterFile_size is called when entering the file_size production.
	EnterFile_size(c *File_sizeContext)

	// ExitTsql_file is called when exiting the tsql_file production.
	ExitTsql_file(c *Tsql_fileContext)

	// ExitBatch is called when exiting the batch production.
	ExitBatch(c *BatchContext)

	// ExitBatch_level_statement is called when exiting the batch_level_statement production.
	ExitBatch_level_statement(c *Batch_level_statementContext)

	// ExitSql_clauses is called when exiting the sql_clauses production.
	ExitSql_clauses(c *Sql_clausesContext)

	// ExitDml_clause is called when exiting the dml_clause production.
	ExitDml_clause(c *Dml_clauseContext)

	// ExitDdl_clause is called when exiting the ddl_clause production.
	ExitDdl_clause(c *Ddl_clauseContext)

	// ExitBackup_statement is called when exiting the backup_statement production.
	ExitBackup_statement(c *Backup_statementContext)

	// ExitCfl_statement is called when exiting the cfl_statement production.
	ExitCfl_statement(c *Cfl_statementContext)

	// ExitBlock_statement is called when exiting the block_statement production.
	ExitBlock_statement(c *Block_statementContext)

	// ExitBreak_statement is called when exiting the break_statement production.
	ExitBreak_statement(c *Break_statementContext)

	// ExitContinue_statement is called when exiting the continue_statement production.
	ExitContinue_statement(c *Continue_statementContext)

	// ExitGoto_statement is called when exiting the goto_statement production.
	ExitGoto_statement(c *Goto_statementContext)

	// ExitReturn_statement is called when exiting the return_statement production.
	ExitReturn_statement(c *Return_statementContext)

	// ExitIf_statement is called when exiting the if_statement production.
	ExitIf_statement(c *If_statementContext)

	// ExitThrow_statement is called when exiting the throw_statement production.
	ExitThrow_statement(c *Throw_statementContext)

	// ExitThrow_error_number is called when exiting the throw_error_number production.
	ExitThrow_error_number(c *Throw_error_numberContext)

	// ExitThrow_message is called when exiting the throw_message production.
	ExitThrow_message(c *Throw_messageContext)

	// ExitThrow_state is called when exiting the throw_state production.
	ExitThrow_state(c *Throw_stateContext)

	// ExitTry_catch_statement is called when exiting the try_catch_statement production.
	ExitTry_catch_statement(c *Try_catch_statementContext)

	// ExitWaitfor_statement is called when exiting the waitfor_statement production.
	ExitWaitfor_statement(c *Waitfor_statementContext)

	// ExitWhile_statement is called when exiting the while_statement production.
	ExitWhile_statement(c *While_statementContext)

	// ExitPrint_statement is called when exiting the print_statement production.
	ExitPrint_statement(c *Print_statementContext)

	// ExitRaiseerror_statement is called when exiting the raiseerror_statement production.
	ExitRaiseerror_statement(c *Raiseerror_statementContext)

	// ExitEmpty_statement is called when exiting the empty_statement production.
	ExitEmpty_statement(c *Empty_statementContext)

	// ExitAnother_statement is called when exiting the another_statement production.
	ExitAnother_statement(c *Another_statementContext)

	// ExitAlter_application_role is called when exiting the alter_application_role production.
	ExitAlter_application_role(c *Alter_application_roleContext)

	// ExitAlter_xml_schema_collection is called when exiting the alter_xml_schema_collection production.
	ExitAlter_xml_schema_collection(c *Alter_xml_schema_collectionContext)

	// ExitCreate_application_role is called when exiting the create_application_role production.
	ExitCreate_application_role(c *Create_application_roleContext)

	// ExitDrop_aggregate is called when exiting the drop_aggregate production.
	ExitDrop_aggregate(c *Drop_aggregateContext)

	// ExitDrop_application_role is called when exiting the drop_application_role production.
	ExitDrop_application_role(c *Drop_application_roleContext)

	// ExitAlter_assembly is called when exiting the alter_assembly production.
	ExitAlter_assembly(c *Alter_assemblyContext)

	// ExitAlter_assembly_start is called when exiting the alter_assembly_start production.
	ExitAlter_assembly_start(c *Alter_assembly_startContext)

	// ExitAlter_assembly_clause is called when exiting the alter_assembly_clause production.
	ExitAlter_assembly_clause(c *Alter_assembly_clauseContext)

	// ExitAlter_assembly_from_clause is called when exiting the alter_assembly_from_clause production.
	ExitAlter_assembly_from_clause(c *Alter_assembly_from_clauseContext)

	// ExitAlter_assembly_from_clause_start is called when exiting the alter_assembly_from_clause_start production.
	ExitAlter_assembly_from_clause_start(c *Alter_assembly_from_clause_startContext)

	// ExitAlter_assembly_drop_clause is called when exiting the alter_assembly_drop_clause production.
	ExitAlter_assembly_drop_clause(c *Alter_assembly_drop_clauseContext)

	// ExitAlter_assembly_drop_multiple_files is called when exiting the alter_assembly_drop_multiple_files production.
	ExitAlter_assembly_drop_multiple_files(c *Alter_assembly_drop_multiple_filesContext)

	// ExitAlter_assembly_drop is called when exiting the alter_assembly_drop production.
	ExitAlter_assembly_drop(c *Alter_assembly_dropContext)

	// ExitAlter_assembly_add_clause is called when exiting the alter_assembly_add_clause production.
	ExitAlter_assembly_add_clause(c *Alter_assembly_add_clauseContext)

	// ExitAlter_asssembly_add_clause_start is called when exiting the alter_asssembly_add_clause_start production.
	ExitAlter_asssembly_add_clause_start(c *Alter_asssembly_add_clause_startContext)

	// ExitAlter_assembly_client_file_clause is called when exiting the alter_assembly_client_file_clause production.
	ExitAlter_assembly_client_file_clause(c *Alter_assembly_client_file_clauseContext)

	// ExitAlter_assembly_file_name is called when exiting the alter_assembly_file_name production.
	ExitAlter_assembly_file_name(c *Alter_assembly_file_nameContext)

	// ExitAlter_assembly_file_bits is called when exiting the alter_assembly_file_bits production.
	ExitAlter_assembly_file_bits(c *Alter_assembly_file_bitsContext)

	// ExitAlter_assembly_as is called when exiting the alter_assembly_as production.
	ExitAlter_assembly_as(c *Alter_assembly_asContext)

	// ExitAlter_assembly_with_clause is called when exiting the alter_assembly_with_clause production.
	ExitAlter_assembly_with_clause(c *Alter_assembly_with_clauseContext)

	// ExitAlter_assembly_with is called when exiting the alter_assembly_with production.
	ExitAlter_assembly_with(c *Alter_assembly_withContext)

	// ExitClient_assembly_specifier is called when exiting the client_assembly_specifier production.
	ExitClient_assembly_specifier(c *Client_assembly_specifierContext)

	// ExitAssembly_option is called when exiting the assembly_option production.
	ExitAssembly_option(c *Assembly_optionContext)

	// ExitNetwork_file_share is called when exiting the network_file_share production.
	ExitNetwork_file_share(c *Network_file_shareContext)

	// ExitNetwork_computer is called when exiting the network_computer production.
	ExitNetwork_computer(c *Network_computerContext)

	// ExitNetwork_file_start is called when exiting the network_file_start production.
	ExitNetwork_file_start(c *Network_file_startContext)

	// ExitFile_path is called when exiting the file_path production.
	ExitFile_path(c *File_pathContext)

	// ExitFile_directory_path_separator is called when exiting the file_directory_path_separator production.
	ExitFile_directory_path_separator(c *File_directory_path_separatorContext)

	// ExitLocal_file is called when exiting the local_file production.
	ExitLocal_file(c *Local_fileContext)

	// ExitLocal_drive is called when exiting the local_drive production.
	ExitLocal_drive(c *Local_driveContext)

	// ExitMultiple_local_files is called when exiting the multiple_local_files production.
	ExitMultiple_local_files(c *Multiple_local_filesContext)

	// ExitMultiple_local_file_start is called when exiting the multiple_local_file_start production.
	ExitMultiple_local_file_start(c *Multiple_local_file_startContext)

	// ExitCreate_assembly is called when exiting the create_assembly production.
	ExitCreate_assembly(c *Create_assemblyContext)

	// ExitDrop_assembly is called when exiting the drop_assembly production.
	ExitDrop_assembly(c *Drop_assemblyContext)

	// ExitAlter_asymmetric_key is called when exiting the alter_asymmetric_key production.
	ExitAlter_asymmetric_key(c *Alter_asymmetric_keyContext)

	// ExitAlter_asymmetric_key_start is called when exiting the alter_asymmetric_key_start production.
	ExitAlter_asymmetric_key_start(c *Alter_asymmetric_key_startContext)

	// ExitAsymmetric_key_option is called when exiting the asymmetric_key_option production.
	ExitAsymmetric_key_option(c *Asymmetric_key_optionContext)

	// ExitAsymmetric_key_option_start is called when exiting the asymmetric_key_option_start production.
	ExitAsymmetric_key_option_start(c *Asymmetric_key_option_startContext)

	// ExitAsymmetric_key_password_change_option is called when exiting the asymmetric_key_password_change_option production.
	ExitAsymmetric_key_password_change_option(c *Asymmetric_key_password_change_optionContext)

	// ExitCreate_asymmetric_key is called when exiting the create_asymmetric_key production.
	ExitCreate_asymmetric_key(c *Create_asymmetric_keyContext)

	// ExitDrop_asymmetric_key is called when exiting the drop_asymmetric_key production.
	ExitDrop_asymmetric_key(c *Drop_asymmetric_keyContext)

	// ExitAlter_authorization is called when exiting the alter_authorization production.
	ExitAlter_authorization(c *Alter_authorizationContext)

	// ExitAuthorization_grantee is called when exiting the authorization_grantee production.
	ExitAuthorization_grantee(c *Authorization_granteeContext)

	// ExitEntity_to is called when exiting the entity_to production.
	ExitEntity_to(c *Entity_toContext)

	// ExitColon_colon is called when exiting the colon_colon production.
	ExitColon_colon(c *Colon_colonContext)

	// ExitAlter_authorization_start is called when exiting the alter_authorization_start production.
	ExitAlter_authorization_start(c *Alter_authorization_startContext)

	// ExitAlter_authorization_for_sql_database is called when exiting the alter_authorization_for_sql_database production.
	ExitAlter_authorization_for_sql_database(c *Alter_authorization_for_sql_databaseContext)

	// ExitAlter_authorization_for_azure_dw is called when exiting the alter_authorization_for_azure_dw production.
	ExitAlter_authorization_for_azure_dw(c *Alter_authorization_for_azure_dwContext)

	// ExitAlter_authorization_for_parallel_dw is called when exiting the alter_authorization_for_parallel_dw production.
	ExitAlter_authorization_for_parallel_dw(c *Alter_authorization_for_parallel_dwContext)

	// ExitClass_type is called when exiting the class_type production.
	ExitClass_type(c *Class_typeContext)

	// ExitClass_type_for_sql_database is called when exiting the class_type_for_sql_database production.
	ExitClass_type_for_sql_database(c *Class_type_for_sql_databaseContext)

	// ExitClass_type_for_azure_dw is called when exiting the class_type_for_azure_dw production.
	ExitClass_type_for_azure_dw(c *Class_type_for_azure_dwContext)

	// ExitClass_type_for_parallel_dw is called when exiting the class_type_for_parallel_dw production.
	ExitClass_type_for_parallel_dw(c *Class_type_for_parallel_dwContext)

	// ExitClass_type_for_grant is called when exiting the class_type_for_grant production.
	ExitClass_type_for_grant(c *Class_type_for_grantContext)

	// ExitDrop_availability_group is called when exiting the drop_availability_group production.
	ExitDrop_availability_group(c *Drop_availability_groupContext)

	// ExitAlter_availability_group is called when exiting the alter_availability_group production.
	ExitAlter_availability_group(c *Alter_availability_groupContext)

	// ExitAlter_availability_group_start is called when exiting the alter_availability_group_start production.
	ExitAlter_availability_group_start(c *Alter_availability_group_startContext)

	// ExitAlter_availability_group_options is called when exiting the alter_availability_group_options production.
	ExitAlter_availability_group_options(c *Alter_availability_group_optionsContext)

	// ExitIp_v4_failover is called when exiting the ip_v4_failover production.
	ExitIp_v4_failover(c *Ip_v4_failoverContext)

	// ExitIp_v6_failover is called when exiting the ip_v6_failover production.
	ExitIp_v6_failover(c *Ip_v6_failoverContext)

	// ExitCreate_or_alter_broker_priority is called when exiting the create_or_alter_broker_priority production.
	ExitCreate_or_alter_broker_priority(c *Create_or_alter_broker_priorityContext)

	// ExitDrop_broker_priority is called when exiting the drop_broker_priority production.
	ExitDrop_broker_priority(c *Drop_broker_priorityContext)

	// ExitAlter_certificate is called when exiting the alter_certificate production.
	ExitAlter_certificate(c *Alter_certificateContext)

	// ExitAlter_column_encryption_key is called when exiting the alter_column_encryption_key production.
	ExitAlter_column_encryption_key(c *Alter_column_encryption_keyContext)

	// ExitCreate_column_encryption_key is called when exiting the create_column_encryption_key production.
	ExitCreate_column_encryption_key(c *Create_column_encryption_keyContext)

	// ExitDrop_certificate is called when exiting the drop_certificate production.
	ExitDrop_certificate(c *Drop_certificateContext)

	// ExitDrop_column_encryption_key is called when exiting the drop_column_encryption_key production.
	ExitDrop_column_encryption_key(c *Drop_column_encryption_keyContext)

	// ExitDrop_column_master_key is called when exiting the drop_column_master_key production.
	ExitDrop_column_master_key(c *Drop_column_master_keyContext)

	// ExitDrop_contract is called when exiting the drop_contract production.
	ExitDrop_contract(c *Drop_contractContext)

	// ExitDrop_credential is called when exiting the drop_credential production.
	ExitDrop_credential(c *Drop_credentialContext)

	// ExitDrop_cryptograhic_provider is called when exiting the drop_cryptograhic_provider production.
	ExitDrop_cryptograhic_provider(c *Drop_cryptograhic_providerContext)

	// ExitDrop_database is called when exiting the drop_database production.
	ExitDrop_database(c *Drop_databaseContext)

	// ExitDrop_database_audit_specification is called when exiting the drop_database_audit_specification production.
	ExitDrop_database_audit_specification(c *Drop_database_audit_specificationContext)

	// ExitDrop_database_encryption_key is called when exiting the drop_database_encryption_key production.
	ExitDrop_database_encryption_key(c *Drop_database_encryption_keyContext)

	// ExitDrop_database_scoped_credential is called when exiting the drop_database_scoped_credential production.
	ExitDrop_database_scoped_credential(c *Drop_database_scoped_credentialContext)

	// ExitDrop_default is called when exiting the drop_default production.
	ExitDrop_default(c *Drop_defaultContext)

	// ExitDrop_endpoint is called when exiting the drop_endpoint production.
	ExitDrop_endpoint(c *Drop_endpointContext)

	// ExitDrop_external_data_source is called when exiting the drop_external_data_source production.
	ExitDrop_external_data_source(c *Drop_external_data_sourceContext)

	// ExitDrop_external_file_format is called when exiting the drop_external_file_format production.
	ExitDrop_external_file_format(c *Drop_external_file_formatContext)

	// ExitDrop_external_library is called when exiting the drop_external_library production.
	ExitDrop_external_library(c *Drop_external_libraryContext)

	// ExitDrop_external_resource_pool is called when exiting the drop_external_resource_pool production.
	ExitDrop_external_resource_pool(c *Drop_external_resource_poolContext)

	// ExitDrop_external_table is called when exiting the drop_external_table production.
	ExitDrop_external_table(c *Drop_external_tableContext)

	// ExitDrop_event_notifications is called when exiting the drop_event_notifications production.
	ExitDrop_event_notifications(c *Drop_event_notificationsContext)

	// ExitDrop_event_session is called when exiting the drop_event_session production.
	ExitDrop_event_session(c *Drop_event_sessionContext)

	// ExitDrop_fulltext_catalog is called when exiting the drop_fulltext_catalog production.
	ExitDrop_fulltext_catalog(c *Drop_fulltext_catalogContext)

	// ExitDrop_fulltext_index is called when exiting the drop_fulltext_index production.
	ExitDrop_fulltext_index(c *Drop_fulltext_indexContext)

	// ExitDrop_fulltext_stoplist is called when exiting the drop_fulltext_stoplist production.
	ExitDrop_fulltext_stoplist(c *Drop_fulltext_stoplistContext)

	// ExitDrop_login is called when exiting the drop_login production.
	ExitDrop_login(c *Drop_loginContext)

	// ExitDrop_master_key is called when exiting the drop_master_key production.
	ExitDrop_master_key(c *Drop_master_keyContext)

	// ExitDrop_message_type is called when exiting the drop_message_type production.
	ExitDrop_message_type(c *Drop_message_typeContext)

	// ExitDrop_partition_function is called when exiting the drop_partition_function production.
	ExitDrop_partition_function(c *Drop_partition_functionContext)

	// ExitDrop_partition_scheme is called when exiting the drop_partition_scheme production.
	ExitDrop_partition_scheme(c *Drop_partition_schemeContext)

	// ExitDrop_queue is called when exiting the drop_queue production.
	ExitDrop_queue(c *Drop_queueContext)

	// ExitDrop_remote_service_binding is called when exiting the drop_remote_service_binding production.
	ExitDrop_remote_service_binding(c *Drop_remote_service_bindingContext)

	// ExitDrop_resource_pool is called when exiting the drop_resource_pool production.
	ExitDrop_resource_pool(c *Drop_resource_poolContext)

	// ExitDrop_db_role is called when exiting the drop_db_role production.
	ExitDrop_db_role(c *Drop_db_roleContext)

	// ExitDrop_route is called when exiting the drop_route production.
	ExitDrop_route(c *Drop_routeContext)

	// ExitDrop_rule is called when exiting the drop_rule production.
	ExitDrop_rule(c *Drop_ruleContext)

	// ExitDrop_schema is called when exiting the drop_schema production.
	ExitDrop_schema(c *Drop_schemaContext)

	// ExitDrop_search_property_list is called when exiting the drop_search_property_list production.
	ExitDrop_search_property_list(c *Drop_search_property_listContext)

	// ExitDrop_security_policy is called when exiting the drop_security_policy production.
	ExitDrop_security_policy(c *Drop_security_policyContext)

	// ExitDrop_sequence is called when exiting the drop_sequence production.
	ExitDrop_sequence(c *Drop_sequenceContext)

	// ExitDrop_server_audit is called when exiting the drop_server_audit production.
	ExitDrop_server_audit(c *Drop_server_auditContext)

	// ExitDrop_server_audit_specification is called when exiting the drop_server_audit_specification production.
	ExitDrop_server_audit_specification(c *Drop_server_audit_specificationContext)

	// ExitDrop_server_role is called when exiting the drop_server_role production.
	ExitDrop_server_role(c *Drop_server_roleContext)

	// ExitDrop_service is called when exiting the drop_service production.
	ExitDrop_service(c *Drop_serviceContext)

	// ExitDrop_signature is called when exiting the drop_signature production.
	ExitDrop_signature(c *Drop_signatureContext)

	// ExitDrop_statistics_name_azure_dw_and_pdw is called when exiting the drop_statistics_name_azure_dw_and_pdw production.
	ExitDrop_statistics_name_azure_dw_and_pdw(c *Drop_statistics_name_azure_dw_and_pdwContext)

	// ExitDrop_symmetric_key is called when exiting the drop_symmetric_key production.
	ExitDrop_symmetric_key(c *Drop_symmetric_keyContext)

	// ExitDrop_synonym is called when exiting the drop_synonym production.
	ExitDrop_synonym(c *Drop_synonymContext)

	// ExitDrop_user is called when exiting the drop_user production.
	ExitDrop_user(c *Drop_userContext)

	// ExitDrop_workload_group is called when exiting the drop_workload_group production.
	ExitDrop_workload_group(c *Drop_workload_groupContext)

	// ExitDrop_xml_schema_collection is called when exiting the drop_xml_schema_collection production.
	ExitDrop_xml_schema_collection(c *Drop_xml_schema_collectionContext)

	// ExitDisable_trigger is called when exiting the disable_trigger production.
	ExitDisable_trigger(c *Disable_triggerContext)

	// ExitEnable_trigger is called when exiting the enable_trigger production.
	ExitEnable_trigger(c *Enable_triggerContext)

	// ExitLock_table is called when exiting the lock_table production.
	ExitLock_table(c *Lock_tableContext)

	// ExitTruncate_table is called when exiting the truncate_table production.
	ExitTruncate_table(c *Truncate_tableContext)

	// ExitCreate_column_master_key is called when exiting the create_column_master_key production.
	ExitCreate_column_master_key(c *Create_column_master_keyContext)

	// ExitAlter_credential is called when exiting the alter_credential production.
	ExitAlter_credential(c *Alter_credentialContext)

	// ExitCreate_credential is called when exiting the create_credential production.
	ExitCreate_credential(c *Create_credentialContext)

	// ExitAlter_cryptographic_provider is called when exiting the alter_cryptographic_provider production.
	ExitAlter_cryptographic_provider(c *Alter_cryptographic_providerContext)

	// ExitCreate_cryptographic_provider is called when exiting the create_cryptographic_provider production.
	ExitCreate_cryptographic_provider(c *Create_cryptographic_providerContext)

	// ExitCreate_endpoint is called when exiting the create_endpoint production.
	ExitCreate_endpoint(c *Create_endpointContext)

	// ExitEndpoint_encryption_alogorithm_clause is called when exiting the endpoint_encryption_alogorithm_clause production.
	ExitEndpoint_encryption_alogorithm_clause(c *Endpoint_encryption_alogorithm_clauseContext)

	// ExitEndpoint_authentication_clause is called when exiting the endpoint_authentication_clause production.
	ExitEndpoint_authentication_clause(c *Endpoint_authentication_clauseContext)

	// ExitEndpoint_listener_clause is called when exiting the endpoint_listener_clause production.
	ExitEndpoint_listener_clause(c *Endpoint_listener_clauseContext)

	// ExitCreate_event_notification is called when exiting the create_event_notification production.
	ExitCreate_event_notification(c *Create_event_notificationContext)

	// ExitCreate_or_alter_event_session is called when exiting the create_or_alter_event_session production.
	ExitCreate_or_alter_event_session(c *Create_or_alter_event_sessionContext)

	// ExitEvent_session_predicate_expression is called when exiting the event_session_predicate_expression production.
	ExitEvent_session_predicate_expression(c *Event_session_predicate_expressionContext)

	// ExitEvent_session_predicate_factor is called when exiting the event_session_predicate_factor production.
	ExitEvent_session_predicate_factor(c *Event_session_predicate_factorContext)

	// ExitEvent_session_predicate_leaf is called when exiting the event_session_predicate_leaf production.
	ExitEvent_session_predicate_leaf(c *Event_session_predicate_leafContext)

	// ExitAlter_external_data_source is called when exiting the alter_external_data_source production.
	ExitAlter_external_data_source(c *Alter_external_data_sourceContext)

	// ExitAlter_external_library is called when exiting the alter_external_library production.
	ExitAlter_external_library(c *Alter_external_libraryContext)

	// ExitCreate_external_library is called when exiting the create_external_library production.
	ExitCreate_external_library(c *Create_external_libraryContext)

	// ExitAlter_external_resource_pool is called when exiting the alter_external_resource_pool production.
	ExitAlter_external_resource_pool(c *Alter_external_resource_poolContext)

	// ExitCreate_external_resource_pool is called when exiting the create_external_resource_pool production.
	ExitCreate_external_resource_pool(c *Create_external_resource_poolContext)

	// ExitAlter_fulltext_catalog is called when exiting the alter_fulltext_catalog production.
	ExitAlter_fulltext_catalog(c *Alter_fulltext_catalogContext)

	// ExitCreate_fulltext_catalog is called when exiting the create_fulltext_catalog production.
	ExitCreate_fulltext_catalog(c *Create_fulltext_catalogContext)

	// ExitAlter_fulltext_stoplist is called when exiting the alter_fulltext_stoplist production.
	ExitAlter_fulltext_stoplist(c *Alter_fulltext_stoplistContext)

	// ExitCreate_fulltext_stoplist is called when exiting the create_fulltext_stoplist production.
	ExitCreate_fulltext_stoplist(c *Create_fulltext_stoplistContext)

	// ExitAlter_login_sql_server is called when exiting the alter_login_sql_server production.
	ExitAlter_login_sql_server(c *Alter_login_sql_serverContext)

	// ExitCreate_login_sql_server is called when exiting the create_login_sql_server production.
	ExitCreate_login_sql_server(c *Create_login_sql_serverContext)

	// ExitAlter_login_azure_sql is called when exiting the alter_login_azure_sql production.
	ExitAlter_login_azure_sql(c *Alter_login_azure_sqlContext)

	// ExitCreate_login_azure_sql is called when exiting the create_login_azure_sql production.
	ExitCreate_login_azure_sql(c *Create_login_azure_sqlContext)

	// ExitAlter_login_azure_sql_dw_and_pdw is called when exiting the alter_login_azure_sql_dw_and_pdw production.
	ExitAlter_login_azure_sql_dw_and_pdw(c *Alter_login_azure_sql_dw_and_pdwContext)

	// ExitCreate_login_pdw is called when exiting the create_login_pdw production.
	ExitCreate_login_pdw(c *Create_login_pdwContext)

	// ExitAlter_master_key_sql_server is called when exiting the alter_master_key_sql_server production.
	ExitAlter_master_key_sql_server(c *Alter_master_key_sql_serverContext)

	// ExitCreate_master_key_sql_server is called when exiting the create_master_key_sql_server production.
	ExitCreate_master_key_sql_server(c *Create_master_key_sql_serverContext)

	// ExitAlter_master_key_azure_sql is called when exiting the alter_master_key_azure_sql production.
	ExitAlter_master_key_azure_sql(c *Alter_master_key_azure_sqlContext)

	// ExitCreate_master_key_azure_sql is called when exiting the create_master_key_azure_sql production.
	ExitCreate_master_key_azure_sql(c *Create_master_key_azure_sqlContext)

	// ExitAlter_message_type is called when exiting the alter_message_type production.
	ExitAlter_message_type(c *Alter_message_typeContext)

	// ExitAlter_partition_function is called when exiting the alter_partition_function production.
	ExitAlter_partition_function(c *Alter_partition_functionContext)

	// ExitAlter_partition_scheme is called when exiting the alter_partition_scheme production.
	ExitAlter_partition_scheme(c *Alter_partition_schemeContext)

	// ExitAlter_remote_service_binding is called when exiting the alter_remote_service_binding production.
	ExitAlter_remote_service_binding(c *Alter_remote_service_bindingContext)

	// ExitCreate_remote_service_binding is called when exiting the create_remote_service_binding production.
	ExitCreate_remote_service_binding(c *Create_remote_service_bindingContext)

	// ExitCreate_resource_pool is called when exiting the create_resource_pool production.
	ExitCreate_resource_pool(c *Create_resource_poolContext)

	// ExitAlter_resource_governor is called when exiting the alter_resource_governor production.
	ExitAlter_resource_governor(c *Alter_resource_governorContext)

	// ExitAlter_database_audit_specification is called when exiting the alter_database_audit_specification production.
	ExitAlter_database_audit_specification(c *Alter_database_audit_specificationContext)

	// ExitAudit_action_spec_group is called when exiting the audit_action_spec_group production.
	ExitAudit_action_spec_group(c *Audit_action_spec_groupContext)

	// ExitAudit_action_specification is called when exiting the audit_action_specification production.
	ExitAudit_action_specification(c *Audit_action_specificationContext)

	// ExitAction_specification is called when exiting the action_specification production.
	ExitAction_specification(c *Action_specificationContext)

	// ExitAudit_class_name is called when exiting the audit_class_name production.
	ExitAudit_class_name(c *Audit_class_nameContext)

	// ExitAudit_securable is called when exiting the audit_securable production.
	ExitAudit_securable(c *Audit_securableContext)

	// ExitAlter_db_role is called when exiting the alter_db_role production.
	ExitAlter_db_role(c *Alter_db_roleContext)

	// ExitCreate_database_audit_specification is called when exiting the create_database_audit_specification production.
	ExitCreate_database_audit_specification(c *Create_database_audit_specificationContext)

	// ExitCreate_db_role is called when exiting the create_db_role production.
	ExitCreate_db_role(c *Create_db_roleContext)

	// ExitCreate_route is called when exiting the create_route production.
	ExitCreate_route(c *Create_routeContext)

	// ExitCreate_rule is called when exiting the create_rule production.
	ExitCreate_rule(c *Create_ruleContext)

	// ExitAlter_schema_sql is called when exiting the alter_schema_sql production.
	ExitAlter_schema_sql(c *Alter_schema_sqlContext)

	// ExitCreate_schema is called when exiting the create_schema production.
	ExitCreate_schema(c *Create_schemaContext)

	// ExitCreate_schema_azure_sql_dw_and_pdw is called when exiting the create_schema_azure_sql_dw_and_pdw production.
	ExitCreate_schema_azure_sql_dw_and_pdw(c *Create_schema_azure_sql_dw_and_pdwContext)

	// ExitAlter_schema_azure_sql_dw_and_pdw is called when exiting the alter_schema_azure_sql_dw_and_pdw production.
	ExitAlter_schema_azure_sql_dw_and_pdw(c *Alter_schema_azure_sql_dw_and_pdwContext)

	// ExitCreate_search_property_list is called when exiting the create_search_property_list production.
	ExitCreate_search_property_list(c *Create_search_property_listContext)

	// ExitCreate_security_policy is called when exiting the create_security_policy production.
	ExitCreate_security_policy(c *Create_security_policyContext)

	// ExitAlter_sequence is called when exiting the alter_sequence production.
	ExitAlter_sequence(c *Alter_sequenceContext)

	// ExitCreate_sequence is called when exiting the create_sequence production.
	ExitCreate_sequence(c *Create_sequenceContext)

	// ExitAlter_server_audit is called when exiting the alter_server_audit production.
	ExitAlter_server_audit(c *Alter_server_auditContext)

	// ExitCreate_server_audit is called when exiting the create_server_audit production.
	ExitCreate_server_audit(c *Create_server_auditContext)

	// ExitAlter_server_audit_specification is called when exiting the alter_server_audit_specification production.
	ExitAlter_server_audit_specification(c *Alter_server_audit_specificationContext)

	// ExitCreate_server_audit_specification is called when exiting the create_server_audit_specification production.
	ExitCreate_server_audit_specification(c *Create_server_audit_specificationContext)

	// ExitAlter_server_configuration is called when exiting the alter_server_configuration production.
	ExitAlter_server_configuration(c *Alter_server_configurationContext)

	// ExitAlter_server_role is called when exiting the alter_server_role production.
	ExitAlter_server_role(c *Alter_server_roleContext)

	// ExitCreate_server_role is called when exiting the create_server_role production.
	ExitCreate_server_role(c *Create_server_roleContext)

	// ExitAlter_server_role_pdw is called when exiting the alter_server_role_pdw production.
	ExitAlter_server_role_pdw(c *Alter_server_role_pdwContext)

	// ExitAlter_service is called when exiting the alter_service production.
	ExitAlter_service(c *Alter_serviceContext)

	// ExitOpt_arg_clause is called when exiting the opt_arg_clause production.
	ExitOpt_arg_clause(c *Opt_arg_clauseContext)

	// ExitCreate_service is called when exiting the create_service production.
	ExitCreate_service(c *Create_serviceContext)

	// ExitAlter_service_master_key is called when exiting the alter_service_master_key production.
	ExitAlter_service_master_key(c *Alter_service_master_keyContext)

	// ExitAlter_symmetric_key is called when exiting the alter_symmetric_key production.
	ExitAlter_symmetric_key(c *Alter_symmetric_keyContext)

	// ExitCreate_synonym is called when exiting the create_synonym production.
	ExitCreate_synonym(c *Create_synonymContext)

	// ExitAlter_user is called when exiting the alter_user production.
	ExitAlter_user(c *Alter_userContext)

	// ExitCreate_user is called when exiting the create_user production.
	ExitCreate_user(c *Create_userContext)

	// ExitCreate_user_azure_sql_dw is called when exiting the create_user_azure_sql_dw production.
	ExitCreate_user_azure_sql_dw(c *Create_user_azure_sql_dwContext)

	// ExitAlter_user_azure_sql is called when exiting the alter_user_azure_sql production.
	ExitAlter_user_azure_sql(c *Alter_user_azure_sqlContext)

	// ExitAlter_workload_group is called when exiting the alter_workload_group production.
	ExitAlter_workload_group(c *Alter_workload_groupContext)

	// ExitCreate_workload_group is called when exiting the create_workload_group production.
	ExitCreate_workload_group(c *Create_workload_groupContext)

	// ExitCreate_xml_schema_collection is called when exiting the create_xml_schema_collection production.
	ExitCreate_xml_schema_collection(c *Create_xml_schema_collectionContext)

	// ExitCreate_partition_function is called when exiting the create_partition_function production.
	ExitCreate_partition_function(c *Create_partition_functionContext)

	// ExitCreate_partition_scheme is called when exiting the create_partition_scheme production.
	ExitCreate_partition_scheme(c *Create_partition_schemeContext)

	// ExitCreate_queue is called when exiting the create_queue production.
	ExitCreate_queue(c *Create_queueContext)

	// ExitQueue_settings is called when exiting the queue_settings production.
	ExitQueue_settings(c *Queue_settingsContext)

	// ExitAlter_queue is called when exiting the alter_queue production.
	ExitAlter_queue(c *Alter_queueContext)

	// ExitQueue_action is called when exiting the queue_action production.
	ExitQueue_action(c *Queue_actionContext)

	// ExitQueue_rebuild_options is called when exiting the queue_rebuild_options production.
	ExitQueue_rebuild_options(c *Queue_rebuild_optionsContext)

	// ExitCreate_contract is called when exiting the create_contract production.
	ExitCreate_contract(c *Create_contractContext)

	// ExitConversation_statement is called when exiting the conversation_statement production.
	ExitConversation_statement(c *Conversation_statementContext)

	// ExitMessage_statement is called when exiting the message_statement production.
	ExitMessage_statement(c *Message_statementContext)

	// ExitMerge_statement is called when exiting the merge_statement production.
	ExitMerge_statement(c *Merge_statementContext)

	// ExitWhen_matches is called when exiting the when_matches production.
	ExitWhen_matches(c *When_matchesContext)

	// ExitMerge_matched is called when exiting the merge_matched production.
	ExitMerge_matched(c *Merge_matchedContext)

	// ExitMerge_not_matched is called when exiting the merge_not_matched production.
	ExitMerge_not_matched(c *Merge_not_matchedContext)

	// ExitDelete_statement is called when exiting the delete_statement production.
	ExitDelete_statement(c *Delete_statementContext)

	// ExitDelete_statement_from is called when exiting the delete_statement_from production.
	ExitDelete_statement_from(c *Delete_statement_fromContext)

	// ExitInsert_statement is called when exiting the insert_statement production.
	ExitInsert_statement(c *Insert_statementContext)

	// ExitInsert_statement_value is called when exiting the insert_statement_value production.
	ExitInsert_statement_value(c *Insert_statement_valueContext)

	// ExitReceive_statement is called when exiting the receive_statement production.
	ExitReceive_statement(c *Receive_statementContext)

	// ExitSelect_statement_standalone is called when exiting the select_statement_standalone production.
	ExitSelect_statement_standalone(c *Select_statement_standaloneContext)

	// ExitSelect_statement is called when exiting the select_statement production.
	ExitSelect_statement(c *Select_statementContext)

	// ExitTime is called when exiting the time production.
	ExitTime(c *TimeContext)

	// ExitUpdate_statement is called when exiting the update_statement production.
	ExitUpdate_statement(c *Update_statementContext)

	// ExitOutput_clause is called when exiting the output_clause production.
	ExitOutput_clause(c *Output_clauseContext)

	// ExitOutput_dml_list_elem is called when exiting the output_dml_list_elem production.
	ExitOutput_dml_list_elem(c *Output_dml_list_elemContext)

	// ExitCreate_database is called when exiting the create_database production.
	ExitCreate_database(c *Create_databaseContext)

	// ExitCreate_index is called when exiting the create_index production.
	ExitCreate_index(c *Create_indexContext)

	// ExitCreate_index_options is called when exiting the create_index_options production.
	ExitCreate_index_options(c *Create_index_optionsContext)

	// ExitRelational_index_option is called when exiting the relational_index_option production.
	ExitRelational_index_option(c *Relational_index_optionContext)

	// ExitAlter_index is called when exiting the alter_index production.
	ExitAlter_index(c *Alter_indexContext)

	// ExitResumable_index_options is called when exiting the resumable_index_options production.
	ExitResumable_index_options(c *Resumable_index_optionsContext)

	// ExitResumable_index_option is called when exiting the resumable_index_option production.
	ExitResumable_index_option(c *Resumable_index_optionContext)

	// ExitReorganize_partition is called when exiting the reorganize_partition production.
	ExitReorganize_partition(c *Reorganize_partitionContext)

	// ExitReorganize_options is called when exiting the reorganize_options production.
	ExitReorganize_options(c *Reorganize_optionsContext)

	// ExitReorganize_option is called when exiting the reorganize_option production.
	ExitReorganize_option(c *Reorganize_optionContext)

	// ExitSet_index_options is called when exiting the set_index_options production.
	ExitSet_index_options(c *Set_index_optionsContext)

	// ExitSet_index_option is called when exiting the set_index_option production.
	ExitSet_index_option(c *Set_index_optionContext)

	// ExitRebuild_partition is called when exiting the rebuild_partition production.
	ExitRebuild_partition(c *Rebuild_partitionContext)

	// ExitRebuild_index_options is called when exiting the rebuild_index_options production.
	ExitRebuild_index_options(c *Rebuild_index_optionsContext)

	// ExitRebuild_index_option is called when exiting the rebuild_index_option production.
	ExitRebuild_index_option(c *Rebuild_index_optionContext)

	// ExitSingle_partition_rebuild_index_options is called when exiting the single_partition_rebuild_index_options production.
	ExitSingle_partition_rebuild_index_options(c *Single_partition_rebuild_index_optionsContext)

	// ExitSingle_partition_rebuild_index_option is called when exiting the single_partition_rebuild_index_option production.
	ExitSingle_partition_rebuild_index_option(c *Single_partition_rebuild_index_optionContext)

	// ExitOn_partitions is called when exiting the on_partitions production.
	ExitOn_partitions(c *On_partitionsContext)

	// ExitCreate_columnstore_index is called when exiting the create_columnstore_index production.
	ExitCreate_columnstore_index(c *Create_columnstore_indexContext)

	// ExitCreate_columnstore_index_options is called when exiting the create_columnstore_index_options production.
	ExitCreate_columnstore_index_options(c *Create_columnstore_index_optionsContext)

	// ExitColumnstore_index_option is called when exiting the columnstore_index_option production.
	ExitColumnstore_index_option(c *Columnstore_index_optionContext)

	// ExitCreate_nonclustered_columnstore_index is called when exiting the create_nonclustered_columnstore_index production.
	ExitCreate_nonclustered_columnstore_index(c *Create_nonclustered_columnstore_indexContext)

	// ExitCreate_xml_index is called when exiting the create_xml_index production.
	ExitCreate_xml_index(c *Create_xml_indexContext)

	// ExitXml_index_options is called when exiting the xml_index_options production.
	ExitXml_index_options(c *Xml_index_optionsContext)

	// ExitXml_index_option is called when exiting the xml_index_option production.
	ExitXml_index_option(c *Xml_index_optionContext)

	// ExitCreate_or_alter_procedure is called when exiting the create_or_alter_procedure production.
	ExitCreate_or_alter_procedure(c *Create_or_alter_procedureContext)

	// ExitAs_external_name is called when exiting the as_external_name production.
	ExitAs_external_name(c *As_external_nameContext)

	// ExitCreate_or_alter_trigger is called when exiting the create_or_alter_trigger production.
	ExitCreate_or_alter_trigger(c *Create_or_alter_triggerContext)

	// ExitCreate_or_alter_dml_trigger is called when exiting the create_or_alter_dml_trigger production.
	ExitCreate_or_alter_dml_trigger(c *Create_or_alter_dml_triggerContext)

	// ExitDml_trigger_option is called when exiting the dml_trigger_option production.
	ExitDml_trigger_option(c *Dml_trigger_optionContext)

	// ExitDml_trigger_operation is called when exiting the dml_trigger_operation production.
	ExitDml_trigger_operation(c *Dml_trigger_operationContext)

	// ExitCreate_or_alter_ddl_trigger is called when exiting the create_or_alter_ddl_trigger production.
	ExitCreate_or_alter_ddl_trigger(c *Create_or_alter_ddl_triggerContext)

	// ExitDdl_trigger_operation is called when exiting the ddl_trigger_operation production.
	ExitDdl_trigger_operation(c *Ddl_trigger_operationContext)

	// ExitCreate_or_alter_function is called when exiting the create_or_alter_function production.
	ExitCreate_or_alter_function(c *Create_or_alter_functionContext)

	// ExitFunc_body_returns_select is called when exiting the func_body_returns_select production.
	ExitFunc_body_returns_select(c *Func_body_returns_selectContext)

	// ExitFunc_body_returns_table is called when exiting the func_body_returns_table production.
	ExitFunc_body_returns_table(c *Func_body_returns_tableContext)

	// ExitFunc_body_returns_scalar is called when exiting the func_body_returns_scalar production.
	ExitFunc_body_returns_scalar(c *Func_body_returns_scalarContext)

	// ExitProcedure_param_default_value is called when exiting the procedure_param_default_value production.
	ExitProcedure_param_default_value(c *Procedure_param_default_valueContext)

	// ExitProcedure_param is called when exiting the procedure_param production.
	ExitProcedure_param(c *Procedure_paramContext)

	// ExitProcedure_option is called when exiting the procedure_option production.
	ExitProcedure_option(c *Procedure_optionContext)

	// ExitFunction_option is called when exiting the function_option production.
	ExitFunction_option(c *Function_optionContext)

	// ExitCreate_statistics is called when exiting the create_statistics production.
	ExitCreate_statistics(c *Create_statisticsContext)

	// ExitUpdate_statistics is called when exiting the update_statistics production.
	ExitUpdate_statistics(c *Update_statisticsContext)

	// ExitUpdate_statistics_options is called when exiting the update_statistics_options production.
	ExitUpdate_statistics_options(c *Update_statistics_optionsContext)

	// ExitUpdate_statistics_option is called when exiting the update_statistics_option production.
	ExitUpdate_statistics_option(c *Update_statistics_optionContext)

	// ExitCreate_table is called when exiting the create_table production.
	ExitCreate_table(c *Create_tableContext)

	// ExitTable_indices is called when exiting the table_indices production.
	ExitTable_indices(c *Table_indicesContext)

	// ExitTable_options is called when exiting the table_options production.
	ExitTable_options(c *Table_optionsContext)

	// ExitTable_option is called when exiting the table_option production.
	ExitTable_option(c *Table_optionContext)

	// ExitCreate_table_index_options is called when exiting the create_table_index_options production.
	ExitCreate_table_index_options(c *Create_table_index_optionsContext)

	// ExitCreate_table_index_option is called when exiting the create_table_index_option production.
	ExitCreate_table_index_option(c *Create_table_index_optionContext)

	// ExitCreate_view is called when exiting the create_view production.
	ExitCreate_view(c *Create_viewContext)

	// ExitView_attribute is called when exiting the view_attribute production.
	ExitView_attribute(c *View_attributeContext)

	// ExitAlter_table is called when exiting the alter_table production.
	ExitAlter_table(c *Alter_tableContext)

	// ExitSwitch_partition is called when exiting the switch_partition production.
	ExitSwitch_partition(c *Switch_partitionContext)

	// ExitLow_priority_lock_wait is called when exiting the low_priority_lock_wait production.
	ExitLow_priority_lock_wait(c *Low_priority_lock_waitContext)

	// ExitAlter_database is called when exiting the alter_database production.
	ExitAlter_database(c *Alter_databaseContext)

	// ExitAdd_or_modify_files is called when exiting the add_or_modify_files production.
	ExitAdd_or_modify_files(c *Add_or_modify_filesContext)

	// ExitFilespec is called when exiting the filespec production.
	ExitFilespec(c *FilespecContext)

	// ExitAdd_or_modify_filegroups is called when exiting the add_or_modify_filegroups production.
	ExitAdd_or_modify_filegroups(c *Add_or_modify_filegroupsContext)

	// ExitFilegroup_updatability_option is called when exiting the filegroup_updatability_option production.
	ExitFilegroup_updatability_option(c *Filegroup_updatability_optionContext)

	// ExitDatabase_optionspec is called when exiting the database_optionspec production.
	ExitDatabase_optionspec(c *Database_optionspecContext)

	// ExitAuto_option is called when exiting the auto_option production.
	ExitAuto_option(c *Auto_optionContext)

	// ExitChange_tracking_option is called when exiting the change_tracking_option production.
	ExitChange_tracking_option(c *Change_tracking_optionContext)

	// ExitChange_tracking_option_list is called when exiting the change_tracking_option_list production.
	ExitChange_tracking_option_list(c *Change_tracking_option_listContext)

	// ExitContainment_option is called when exiting the containment_option production.
	ExitContainment_option(c *Containment_optionContext)

	// ExitCursor_option is called when exiting the cursor_option production.
	ExitCursor_option(c *Cursor_optionContext)

	// ExitAlter_endpoint is called when exiting the alter_endpoint production.
	ExitAlter_endpoint(c *Alter_endpointContext)

	// ExitDatabase_mirroring_option is called when exiting the database_mirroring_option production.
	ExitDatabase_mirroring_option(c *Database_mirroring_optionContext)

	// ExitMirroring_set_option is called when exiting the mirroring_set_option production.
	ExitMirroring_set_option(c *Mirroring_set_optionContext)

	// ExitMirroring_partner is called when exiting the mirroring_partner production.
	ExitMirroring_partner(c *Mirroring_partnerContext)

	// ExitMirroring_witness is called when exiting the mirroring_witness production.
	ExitMirroring_witness(c *Mirroring_witnessContext)

	// ExitWitness_partner_equal is called when exiting the witness_partner_equal production.
	ExitWitness_partner_equal(c *Witness_partner_equalContext)

	// ExitPartner_option is called when exiting the partner_option production.
	ExitPartner_option(c *Partner_optionContext)

	// ExitWitness_option is called when exiting the witness_option production.
	ExitWitness_option(c *Witness_optionContext)

	// ExitWitness_server is called when exiting the witness_server production.
	ExitWitness_server(c *Witness_serverContext)

	// ExitPartner_server is called when exiting the partner_server production.
	ExitPartner_server(c *Partner_serverContext)

	// ExitMirroring_host_port_seperator is called when exiting the mirroring_host_port_seperator production.
	ExitMirroring_host_port_seperator(c *Mirroring_host_port_seperatorContext)

	// ExitPartner_server_tcp_prefix is called when exiting the partner_server_tcp_prefix production.
	ExitPartner_server_tcp_prefix(c *Partner_server_tcp_prefixContext)

	// ExitPort_number is called when exiting the port_number production.
	ExitPort_number(c *Port_numberContext)

	// ExitHost is called when exiting the host production.
	ExitHost(c *HostContext)

	// ExitDate_correlation_optimization_option is called when exiting the date_correlation_optimization_option production.
	ExitDate_correlation_optimization_option(c *Date_correlation_optimization_optionContext)

	// ExitDb_encryption_option is called when exiting the db_encryption_option production.
	ExitDb_encryption_option(c *Db_encryption_optionContext)

	// ExitDb_state_option is called when exiting the db_state_option production.
	ExitDb_state_option(c *Db_state_optionContext)

	// ExitDb_update_option is called when exiting the db_update_option production.
	ExitDb_update_option(c *Db_update_optionContext)

	// ExitDb_user_access_option is called when exiting the db_user_access_option production.
	ExitDb_user_access_option(c *Db_user_access_optionContext)

	// ExitDelayed_durability_option is called when exiting the delayed_durability_option production.
	ExitDelayed_durability_option(c *Delayed_durability_optionContext)

	// ExitExternal_access_option is called when exiting the external_access_option production.
	ExitExternal_access_option(c *External_access_optionContext)

	// ExitHadr_options is called when exiting the hadr_options production.
	ExitHadr_options(c *Hadr_optionsContext)

	// ExitMixed_page_allocation_option is called when exiting the mixed_page_allocation_option production.
	ExitMixed_page_allocation_option(c *Mixed_page_allocation_optionContext)

	// ExitParameterization_option is called when exiting the parameterization_option production.
	ExitParameterization_option(c *Parameterization_optionContext)

	// ExitRecovery_option is called when exiting the recovery_option production.
	ExitRecovery_option(c *Recovery_optionContext)

	// ExitService_broker_option is called when exiting the service_broker_option production.
	ExitService_broker_option(c *Service_broker_optionContext)

	// ExitSnapshot_option is called when exiting the snapshot_option production.
	ExitSnapshot_option(c *Snapshot_optionContext)

	// ExitSql_option is called when exiting the sql_option production.
	ExitSql_option(c *Sql_optionContext)

	// ExitTarget_recovery_time_option is called when exiting the target_recovery_time_option production.
	ExitTarget_recovery_time_option(c *Target_recovery_time_optionContext)

	// ExitTermination is called when exiting the termination production.
	ExitTermination(c *TerminationContext)

	// ExitDrop_index is called when exiting the drop_index production.
	ExitDrop_index(c *Drop_indexContext)

	// ExitDrop_relational_or_xml_or_spatial_index is called when exiting the drop_relational_or_xml_or_spatial_index production.
	ExitDrop_relational_or_xml_or_spatial_index(c *Drop_relational_or_xml_or_spatial_indexContext)

	// ExitDrop_backward_compatible_index is called when exiting the drop_backward_compatible_index production.
	ExitDrop_backward_compatible_index(c *Drop_backward_compatible_indexContext)

	// ExitDrop_procedure is called when exiting the drop_procedure production.
	ExitDrop_procedure(c *Drop_procedureContext)

	// ExitDrop_trigger is called when exiting the drop_trigger production.
	ExitDrop_trigger(c *Drop_triggerContext)

	// ExitDrop_dml_trigger is called when exiting the drop_dml_trigger production.
	ExitDrop_dml_trigger(c *Drop_dml_triggerContext)

	// ExitDrop_ddl_trigger is called when exiting the drop_ddl_trigger production.
	ExitDrop_ddl_trigger(c *Drop_ddl_triggerContext)

	// ExitDrop_function is called when exiting the drop_function production.
	ExitDrop_function(c *Drop_functionContext)

	// ExitDrop_statistics is called when exiting the drop_statistics production.
	ExitDrop_statistics(c *Drop_statisticsContext)

	// ExitDrop_table is called when exiting the drop_table production.
	ExitDrop_table(c *Drop_tableContext)

	// ExitDrop_view is called when exiting the drop_view production.
	ExitDrop_view(c *Drop_viewContext)

	// ExitCreate_type is called when exiting the create_type production.
	ExitCreate_type(c *Create_typeContext)

	// ExitDrop_type is called when exiting the drop_type production.
	ExitDrop_type(c *Drop_typeContext)

	// ExitRowset_function_limited is called when exiting the rowset_function_limited production.
	ExitRowset_function_limited(c *Rowset_function_limitedContext)

	// ExitOpenquery is called when exiting the openquery production.
	ExitOpenquery(c *OpenqueryContext)

	// ExitOpendatasource is called when exiting the opendatasource production.
	ExitOpendatasource(c *OpendatasourceContext)

	// ExitDeclare_statement is called when exiting the declare_statement production.
	ExitDeclare_statement(c *Declare_statementContext)

	// ExitXml_declaration is called when exiting the xml_declaration production.
	ExitXml_declaration(c *Xml_declarationContext)

	// ExitCursor_statement is called when exiting the cursor_statement production.
	ExitCursor_statement(c *Cursor_statementContext)

	// ExitBackup_database is called when exiting the backup_database production.
	ExitBackup_database(c *Backup_databaseContext)

	// ExitBackup_log is called when exiting the backup_log production.
	ExitBackup_log(c *Backup_logContext)

	// ExitBackup_certificate is called when exiting the backup_certificate production.
	ExitBackup_certificate(c *Backup_certificateContext)

	// ExitBackup_master_key is called when exiting the backup_master_key production.
	ExitBackup_master_key(c *Backup_master_keyContext)

	// ExitBackup_service_master_key is called when exiting the backup_service_master_key production.
	ExitBackup_service_master_key(c *Backup_service_master_keyContext)

	// ExitKill_statement is called when exiting the kill_statement production.
	ExitKill_statement(c *Kill_statementContext)

	// ExitKill_process is called when exiting the kill_process production.
	ExitKill_process(c *Kill_processContext)

	// ExitKill_query_notification is called when exiting the kill_query_notification production.
	ExitKill_query_notification(c *Kill_query_notificationContext)

	// ExitKill_stats_job is called when exiting the kill_stats_job production.
	ExitKill_stats_job(c *Kill_stats_jobContext)

	// ExitExecute_statement is called when exiting the execute_statement production.
	ExitExecute_statement(c *Execute_statementContext)

	// ExitExecute_body_batch is called when exiting the execute_body_batch production.
	ExitExecute_body_batch(c *Execute_body_batchContext)

	// ExitExecute_body is called when exiting the execute_body production.
	ExitExecute_body(c *Execute_bodyContext)

	// ExitExecute_statement_arg is called when exiting the execute_statement_arg production.
	ExitExecute_statement_arg(c *Execute_statement_argContext)

	// ExitExecute_statement_arg_named is called when exiting the execute_statement_arg_named production.
	ExitExecute_statement_arg_named(c *Execute_statement_arg_namedContext)

	// ExitExecute_statement_arg_unnamed is called when exiting the execute_statement_arg_unnamed production.
	ExitExecute_statement_arg_unnamed(c *Execute_statement_arg_unnamedContext)

	// ExitExecute_parameter is called when exiting the execute_parameter production.
	ExitExecute_parameter(c *Execute_parameterContext)

	// ExitExecute_var_string is called when exiting the execute_var_string production.
	ExitExecute_var_string(c *Execute_var_stringContext)

	// ExitSecurity_statement is called when exiting the security_statement production.
	ExitSecurity_statement(c *Security_statementContext)

	// ExitPrincipal_id is called when exiting the principal_id production.
	ExitPrincipal_id(c *Principal_idContext)

	// ExitCreate_certificate is called when exiting the create_certificate production.
	ExitCreate_certificate(c *Create_certificateContext)

	// ExitExisting_keys is called when exiting the existing_keys production.
	ExitExisting_keys(c *Existing_keysContext)

	// ExitPrivate_key_options is called when exiting the private_key_options production.
	ExitPrivate_key_options(c *Private_key_optionsContext)

	// ExitGenerate_new_keys is called when exiting the generate_new_keys production.
	ExitGenerate_new_keys(c *Generate_new_keysContext)

	// ExitDate_options is called when exiting the date_options production.
	ExitDate_options(c *Date_optionsContext)

	// ExitOpen_key is called when exiting the open_key production.
	ExitOpen_key(c *Open_keyContext)

	// ExitClose_key is called when exiting the close_key production.
	ExitClose_key(c *Close_keyContext)

	// ExitCreate_key is called when exiting the create_key production.
	ExitCreate_key(c *Create_keyContext)

	// ExitKey_options is called when exiting the key_options production.
	ExitKey_options(c *Key_optionsContext)

	// ExitAlgorithm is called when exiting the algorithm production.
	ExitAlgorithm(c *AlgorithmContext)

	// ExitEncryption_mechanism is called when exiting the encryption_mechanism production.
	ExitEncryption_mechanism(c *Encryption_mechanismContext)

	// ExitDecryption_mechanism is called when exiting the decryption_mechanism production.
	ExitDecryption_mechanism(c *Decryption_mechanismContext)

	// ExitGrant_permission is called when exiting the grant_permission production.
	ExitGrant_permission(c *Grant_permissionContext)

	// ExitSet_statement is called when exiting the set_statement production.
	ExitSet_statement(c *Set_statementContext)

	// ExitTransaction_statement is called when exiting the transaction_statement production.
	ExitTransaction_statement(c *Transaction_statementContext)

	// ExitGo_statement is called when exiting the go_statement production.
	ExitGo_statement(c *Go_statementContext)

	// ExitUse_statement is called when exiting the use_statement production.
	ExitUse_statement(c *Use_statementContext)

	// ExitSetuser_statement is called when exiting the setuser_statement production.
	ExitSetuser_statement(c *Setuser_statementContext)

	// ExitReconfigure_statement is called when exiting the reconfigure_statement production.
	ExitReconfigure_statement(c *Reconfigure_statementContext)

	// ExitShutdown_statement is called when exiting the shutdown_statement production.
	ExitShutdown_statement(c *Shutdown_statementContext)

	// ExitCheckpoint_statement is called when exiting the checkpoint_statement production.
	ExitCheckpoint_statement(c *Checkpoint_statementContext)

	// ExitDbcc_checkalloc_option is called when exiting the dbcc_checkalloc_option production.
	ExitDbcc_checkalloc_option(c *Dbcc_checkalloc_optionContext)

	// ExitDbcc_checkalloc is called when exiting the dbcc_checkalloc production.
	ExitDbcc_checkalloc(c *Dbcc_checkallocContext)

	// ExitDbcc_checkcatalog is called when exiting the dbcc_checkcatalog production.
	ExitDbcc_checkcatalog(c *Dbcc_checkcatalogContext)

	// ExitDbcc_checkconstraints_option is called when exiting the dbcc_checkconstraints_option production.
	ExitDbcc_checkconstraints_option(c *Dbcc_checkconstraints_optionContext)

	// ExitDbcc_checkconstraints is called when exiting the dbcc_checkconstraints production.
	ExitDbcc_checkconstraints(c *Dbcc_checkconstraintsContext)

	// ExitDbcc_checkdb_table_option is called when exiting the dbcc_checkdb_table_option production.
	ExitDbcc_checkdb_table_option(c *Dbcc_checkdb_table_optionContext)

	// ExitDbcc_checkdb is called when exiting the dbcc_checkdb production.
	ExitDbcc_checkdb(c *Dbcc_checkdbContext)

	// ExitDbcc_checkfilegroup_option is called when exiting the dbcc_checkfilegroup_option production.
	ExitDbcc_checkfilegroup_option(c *Dbcc_checkfilegroup_optionContext)

	// ExitDbcc_checkfilegroup is called when exiting the dbcc_checkfilegroup production.
	ExitDbcc_checkfilegroup(c *Dbcc_checkfilegroupContext)

	// ExitDbcc_checktable is called when exiting the dbcc_checktable production.
	ExitDbcc_checktable(c *Dbcc_checktableContext)

	// ExitDbcc_cleantable is called when exiting the dbcc_cleantable production.
	ExitDbcc_cleantable(c *Dbcc_cleantableContext)

	// ExitDbcc_clonedatabase_option is called when exiting the dbcc_clonedatabase_option production.
	ExitDbcc_clonedatabase_option(c *Dbcc_clonedatabase_optionContext)

	// ExitDbcc_clonedatabase is called when exiting the dbcc_clonedatabase production.
	ExitDbcc_clonedatabase(c *Dbcc_clonedatabaseContext)

	// ExitDbcc_pdw_showspaceused is called when exiting the dbcc_pdw_showspaceused production.
	ExitDbcc_pdw_showspaceused(c *Dbcc_pdw_showspaceusedContext)

	// ExitDbcc_proccache is called when exiting the dbcc_proccache production.
	ExitDbcc_proccache(c *Dbcc_proccacheContext)

	// ExitDbcc_showcontig_option is called when exiting the dbcc_showcontig_option production.
	ExitDbcc_showcontig_option(c *Dbcc_showcontig_optionContext)

	// ExitDbcc_showcontig is called when exiting the dbcc_showcontig production.
	ExitDbcc_showcontig(c *Dbcc_showcontigContext)

	// ExitDbcc_shrinklog is called when exiting the dbcc_shrinklog production.
	ExitDbcc_shrinklog(c *Dbcc_shrinklogContext)

	// ExitDbcc_dbreindex is called when exiting the dbcc_dbreindex production.
	ExitDbcc_dbreindex(c *Dbcc_dbreindexContext)

	// ExitDbcc_dll_free is called when exiting the dbcc_dll_free production.
	ExitDbcc_dll_free(c *Dbcc_dll_freeContext)

	// ExitDbcc_dropcleanbuffers is called when exiting the dbcc_dropcleanbuffers production.
	ExitDbcc_dropcleanbuffers(c *Dbcc_dropcleanbuffersContext)

	// ExitDbcc_clause is called when exiting the dbcc_clause production.
	ExitDbcc_clause(c *Dbcc_clauseContext)

	// ExitExecute_clause is called when exiting the execute_clause production.
	ExitExecute_clause(c *Execute_clauseContext)

	// ExitDeclare_local is called when exiting the declare_local production.
	ExitDeclare_local(c *Declare_localContext)

	// ExitTable_type_definition is called when exiting the table_type_definition production.
	ExitTable_type_definition(c *Table_type_definitionContext)

	// ExitTable_type_indices is called when exiting the table_type_indices production.
	ExitTable_type_indices(c *Table_type_indicesContext)

	// ExitXml_type_definition is called when exiting the xml_type_definition production.
	ExitXml_type_definition(c *Xml_type_definitionContext)

	// ExitXml_schema_collection is called when exiting the xml_schema_collection production.
	ExitXml_schema_collection(c *Xml_schema_collectionContext)

	// ExitColumn_def_table_constraints is called when exiting the column_def_table_constraints production.
	ExitColumn_def_table_constraints(c *Column_def_table_constraintsContext)

	// ExitColumn_def_table_constraint is called when exiting the column_def_table_constraint production.
	ExitColumn_def_table_constraint(c *Column_def_table_constraintContext)

	// ExitColumn_definition is called when exiting the column_definition production.
	ExitColumn_definition(c *Column_definitionContext)

	// ExitColumn_definition_element is called when exiting the column_definition_element production.
	ExitColumn_definition_element(c *Column_definition_elementContext)

	// ExitColumn_modifier is called when exiting the column_modifier production.
	ExitColumn_modifier(c *Column_modifierContext)

	// ExitMaterialized_column_definition is called when exiting the materialized_column_definition production.
	ExitMaterialized_column_definition(c *Materialized_column_definitionContext)

	// ExitColumn_constraint is called when exiting the column_constraint production.
	ExitColumn_constraint(c *Column_constraintContext)

	// ExitColumn_index is called when exiting the column_index production.
	ExitColumn_index(c *Column_indexContext)

	// ExitOn_partition_or_filegroup is called when exiting the on_partition_or_filegroup production.
	ExitOn_partition_or_filegroup(c *On_partition_or_filegroupContext)

	// ExitTable_constraint is called when exiting the table_constraint production.
	ExitTable_constraint(c *Table_constraintContext)

	// ExitConnection_node is called when exiting the connection_node production.
	ExitConnection_node(c *Connection_nodeContext)

	// ExitPrimary_key_options is called when exiting the primary_key_options production.
	ExitPrimary_key_options(c *Primary_key_optionsContext)

	// ExitForeign_key_options is called when exiting the foreign_key_options production.
	ExitForeign_key_options(c *Foreign_key_optionsContext)

	// ExitCheck_constraint is called when exiting the check_constraint production.
	ExitCheck_constraint(c *Check_constraintContext)

	// ExitOn_delete is called when exiting the on_delete production.
	ExitOn_delete(c *On_deleteContext)

	// ExitOn_update is called when exiting the on_update production.
	ExitOn_update(c *On_updateContext)

	// ExitAlter_table_index_options is called when exiting the alter_table_index_options production.
	ExitAlter_table_index_options(c *Alter_table_index_optionsContext)

	// ExitAlter_table_index_option is called when exiting the alter_table_index_option production.
	ExitAlter_table_index_option(c *Alter_table_index_optionContext)

	// ExitDeclare_cursor is called when exiting the declare_cursor production.
	ExitDeclare_cursor(c *Declare_cursorContext)

	// ExitDeclare_set_cursor_common is called when exiting the declare_set_cursor_common production.
	ExitDeclare_set_cursor_common(c *Declare_set_cursor_commonContext)

	// ExitDeclare_set_cursor_common_partial is called when exiting the declare_set_cursor_common_partial production.
	ExitDeclare_set_cursor_common_partial(c *Declare_set_cursor_common_partialContext)

	// ExitFetch_cursor is called when exiting the fetch_cursor production.
	ExitFetch_cursor(c *Fetch_cursorContext)

	// ExitSet_special is called when exiting the set_special production.
	ExitSet_special(c *Set_specialContext)

	// ExitSpecial_list is called when exiting the special_list production.
	ExitSpecial_list(c *Special_listContext)

	// ExitConstant_LOCAL_ID is called when exiting the constant_LOCAL_ID production.
	ExitConstant_LOCAL_ID(c *Constant_LOCAL_IDContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitParameter is called when exiting the parameter production.
	ExitParameter(c *ParameterContext)

	// ExitTime_zone is called when exiting the time_zone production.
	ExitTime_zone(c *Time_zoneContext)

	// ExitPrimitive_expression is called when exiting the primitive_expression production.
	ExitPrimitive_expression(c *Primitive_expressionContext)

	// ExitCase_expression is called when exiting the case_expression production.
	ExitCase_expression(c *Case_expressionContext)

	// ExitUnary_operator_expression is called when exiting the unary_operator_expression production.
	ExitUnary_operator_expression(c *Unary_operator_expressionContext)

	// ExitBracket_expression is called when exiting the bracket_expression production.
	ExitBracket_expression(c *Bracket_expressionContext)

	// ExitSubquery is called when exiting the subquery production.
	ExitSubquery(c *SubqueryContext)

	// ExitWith_expression is called when exiting the with_expression production.
	ExitWith_expression(c *With_expressionContext)

	// ExitCommon_table_expression is called when exiting the common_table_expression production.
	ExitCommon_table_expression(c *Common_table_expressionContext)

	// ExitUpdate_elem is called when exiting the update_elem production.
	ExitUpdate_elem(c *Update_elemContext)

	// ExitUpdate_elem_merge is called when exiting the update_elem_merge production.
	ExitUpdate_elem_merge(c *Update_elem_mergeContext)

	// ExitSearch_condition is called when exiting the search_condition production.
	ExitSearch_condition(c *Search_conditionContext)

	// ExitPredicate is called when exiting the predicate production.
	ExitPredicate(c *PredicateContext)

	// ExitQuery_expression is called when exiting the query_expression production.
	ExitQuery_expression(c *Query_expressionContext)

	// ExitSql_union is called when exiting the sql_union production.
	ExitSql_union(c *Sql_unionContext)

	// ExitQuery_specification is called when exiting the query_specification production.
	ExitQuery_specification(c *Query_specificationContext)

	// ExitTop_clause is called when exiting the top_clause production.
	ExitTop_clause(c *Top_clauseContext)

	// ExitTop_percent is called when exiting the top_percent production.
	ExitTop_percent(c *Top_percentContext)

	// ExitTop_count is called when exiting the top_count production.
	ExitTop_count(c *Top_countContext)

	// ExitOrder_by_clause is called when exiting the order_by_clause production.
	ExitOrder_by_clause(c *Order_by_clauseContext)

	// ExitSelect_order_by_clause is called when exiting the select_order_by_clause production.
	ExitSelect_order_by_clause(c *Select_order_by_clauseContext)

	// ExitFor_clause is called when exiting the for_clause production.
	ExitFor_clause(c *For_clauseContext)

	// ExitXml_common_directives is called when exiting the xml_common_directives production.
	ExitXml_common_directives(c *Xml_common_directivesContext)

	// ExitOrder_by_expression is called when exiting the order_by_expression production.
	ExitOrder_by_expression(c *Order_by_expressionContext)

	// ExitGrouping_sets_item is called when exiting the grouping_sets_item production.
	ExitGrouping_sets_item(c *Grouping_sets_itemContext)

	// ExitGroup_by_item is called when exiting the group_by_item production.
	ExitGroup_by_item(c *Group_by_itemContext)

	// ExitOption_clause is called when exiting the option_clause production.
	ExitOption_clause(c *Option_clauseContext)

	// ExitOption is called when exiting the option production.
	ExitOption(c *OptionContext)

	// ExitOptimize_for_arg is called when exiting the optimize_for_arg production.
	ExitOptimize_for_arg(c *Optimize_for_argContext)

	// ExitSelect_list is called when exiting the select_list production.
	ExitSelect_list(c *Select_listContext)

	// ExitUdt_method_arguments is called when exiting the udt_method_arguments production.
	ExitUdt_method_arguments(c *Udt_method_argumentsContext)

	// ExitAsterisk is called when exiting the asterisk production.
	ExitAsterisk(c *AsteriskContext)

	// ExitUdt_elem is called when exiting the udt_elem production.
	ExitUdt_elem(c *Udt_elemContext)

	// ExitExpression_elem is called when exiting the expression_elem production.
	ExitExpression_elem(c *Expression_elemContext)

	// ExitSelect_list_elem is called when exiting the select_list_elem production.
	ExitSelect_list_elem(c *Select_list_elemContext)

	// ExitTable_sources is called when exiting the table_sources production.
	ExitTable_sources(c *Table_sourcesContext)

	// ExitNon_ansi_join is called when exiting the non_ansi_join production.
	ExitNon_ansi_join(c *Non_ansi_joinContext)

	// ExitTable_source is called when exiting the table_source production.
	ExitTable_source(c *Table_sourceContext)

	// ExitTable_source_item is called when exiting the table_source_item production.
	ExitTable_source_item(c *Table_source_itemContext)

	// ExitOpen_xml is called when exiting the open_xml production.
	ExitOpen_xml(c *Open_xmlContext)

	// ExitOpen_json is called when exiting the open_json production.
	ExitOpen_json(c *Open_jsonContext)

	// ExitJson_declaration is called when exiting the json_declaration production.
	ExitJson_declaration(c *Json_declarationContext)

	// ExitJson_column_declaration is called when exiting the json_column_declaration production.
	ExitJson_column_declaration(c *Json_column_declarationContext)

	// ExitSchema_declaration is called when exiting the schema_declaration production.
	ExitSchema_declaration(c *Schema_declarationContext)

	// ExitColumn_declaration is called when exiting the column_declaration production.
	ExitColumn_declaration(c *Column_declarationContext)

	// ExitChange_table is called when exiting the change_table production.
	ExitChange_table(c *Change_tableContext)

	// ExitChange_table_changes is called when exiting the change_table_changes production.
	ExitChange_table_changes(c *Change_table_changesContext)

	// ExitChange_table_version is called when exiting the change_table_version production.
	ExitChange_table_version(c *Change_table_versionContext)

	// ExitJoin_part is called when exiting the join_part production.
	ExitJoin_part(c *Join_partContext)

	// ExitJoin_on is called when exiting the join_on production.
	ExitJoin_on(c *Join_onContext)

	// ExitCross_join is called when exiting the cross_join production.
	ExitCross_join(c *Cross_joinContext)

	// ExitApply_ is called when exiting the apply_ production.
	ExitApply_(c *Apply_Context)

	// ExitPivot is called when exiting the pivot production.
	ExitPivot(c *PivotContext)

	// ExitUnpivot is called when exiting the unpivot production.
	ExitUnpivot(c *UnpivotContext)

	// ExitPivot_clause is called when exiting the pivot_clause production.
	ExitPivot_clause(c *Pivot_clauseContext)

	// ExitUnpivot_clause is called when exiting the unpivot_clause production.
	ExitUnpivot_clause(c *Unpivot_clauseContext)

	// ExitFull_column_name_list is called when exiting the full_column_name_list production.
	ExitFull_column_name_list(c *Full_column_name_listContext)

	// ExitRowset_function is called when exiting the rowset_function production.
	ExitRowset_function(c *Rowset_functionContext)

	// ExitBulk_option is called when exiting the bulk_option production.
	ExitBulk_option(c *Bulk_optionContext)

	// ExitDerived_table is called when exiting the derived_table production.
	ExitDerived_table(c *Derived_tableContext)

	// ExitRANKING_WINDOWED_FUNC is called when exiting the RANKING_WINDOWED_FUNC production.
	ExitRANKING_WINDOWED_FUNC(c *RANKING_WINDOWED_FUNCContext)

	// ExitAGGREGATE_WINDOWED_FUNC is called when exiting the AGGREGATE_WINDOWED_FUNC production.
	ExitAGGREGATE_WINDOWED_FUNC(c *AGGREGATE_WINDOWED_FUNCContext)

	// ExitANALYTIC_WINDOWED_FUNC is called when exiting the ANALYTIC_WINDOWED_FUNC production.
	ExitANALYTIC_WINDOWED_FUNC(c *ANALYTIC_WINDOWED_FUNCContext)

	// ExitBUILT_IN_FUNC is called when exiting the BUILT_IN_FUNC production.
	ExitBUILT_IN_FUNC(c *BUILT_IN_FUNCContext)

	// ExitSCALAR_FUNCTION is called when exiting the SCALAR_FUNCTION production.
	ExitSCALAR_FUNCTION(c *SCALAR_FUNCTIONContext)

	// ExitFREE_TEXT is called when exiting the FREE_TEXT production.
	ExitFREE_TEXT(c *FREE_TEXTContext)

	// ExitPARTITION_FUNC is called when exiting the PARTITION_FUNC production.
	ExitPARTITION_FUNC(c *PARTITION_FUNCContext)

	// ExitHIERARCHYID_METHOD is called when exiting the HIERARCHYID_METHOD production.
	ExitHIERARCHYID_METHOD(c *HIERARCHYID_METHODContext)

	// ExitPartition_function is called when exiting the partition_function production.
	ExitPartition_function(c *Partition_functionContext)

	// ExitFreetext_function is called when exiting the freetext_function production.
	ExitFreetext_function(c *Freetext_functionContext)

	// ExitFreetext_predicate is called when exiting the freetext_predicate production.
	ExitFreetext_predicate(c *Freetext_predicateContext)

	// ExitJson_key_value is called when exiting the json_key_value production.
	ExitJson_key_value(c *Json_key_valueContext)

	// ExitJson_null_clause is called when exiting the json_null_clause production.
	ExitJson_null_clause(c *Json_null_clauseContext)

	// ExitAPP_NAME is called when exiting the APP_NAME production.
	ExitAPP_NAME(c *APP_NAMEContext)

	// ExitAPPLOCK_MODE is called when exiting the APPLOCK_MODE production.
	ExitAPPLOCK_MODE(c *APPLOCK_MODEContext)

	// ExitAPPLOCK_TEST is called when exiting the APPLOCK_TEST production.
	ExitAPPLOCK_TEST(c *APPLOCK_TESTContext)

	// ExitASSEMBLYPROPERTY is called when exiting the ASSEMBLYPROPERTY production.
	ExitASSEMBLYPROPERTY(c *ASSEMBLYPROPERTYContext)

	// ExitCOL_LENGTH is called when exiting the COL_LENGTH production.
	ExitCOL_LENGTH(c *COL_LENGTHContext)

	// ExitCOL_NAME is called when exiting the COL_NAME production.
	ExitCOL_NAME(c *COL_NAMEContext)

	// ExitCOLUMNPROPERTY is called when exiting the COLUMNPROPERTY production.
	ExitCOLUMNPROPERTY(c *COLUMNPROPERTYContext)

	// ExitDATABASEPROPERTYEX is called when exiting the DATABASEPROPERTYEX production.
	ExitDATABASEPROPERTYEX(c *DATABASEPROPERTYEXContext)

	// ExitDB_ID is called when exiting the DB_ID production.
	ExitDB_ID(c *DB_IDContext)

	// ExitDB_NAME is called when exiting the DB_NAME production.
	ExitDB_NAME(c *DB_NAMEContext)

	// ExitFILE_ID is called when exiting the FILE_ID production.
	ExitFILE_ID(c *FILE_IDContext)

	// ExitFILE_IDEX is called when exiting the FILE_IDEX production.
	ExitFILE_IDEX(c *FILE_IDEXContext)

	// ExitFILE_NAME is called when exiting the FILE_NAME production.
	ExitFILE_NAME(c *FILE_NAMEContext)

	// ExitFILEGROUP_ID is called when exiting the FILEGROUP_ID production.
	ExitFILEGROUP_ID(c *FILEGROUP_IDContext)

	// ExitFILEGROUP_NAME is called when exiting the FILEGROUP_NAME production.
	ExitFILEGROUP_NAME(c *FILEGROUP_NAMEContext)

	// ExitFILEGROUPPROPERTY is called when exiting the FILEGROUPPROPERTY production.
	ExitFILEGROUPPROPERTY(c *FILEGROUPPROPERTYContext)

	// ExitFILEPROPERTY is called when exiting the FILEPROPERTY production.
	ExitFILEPROPERTY(c *FILEPROPERTYContext)

	// ExitFILEPROPERTYEX is called when exiting the FILEPROPERTYEX production.
	ExitFILEPROPERTYEX(c *FILEPROPERTYEXContext)

	// ExitFULLTEXTCATALOGPROPERTY is called when exiting the FULLTEXTCATALOGPROPERTY production.
	ExitFULLTEXTCATALOGPROPERTY(c *FULLTEXTCATALOGPROPERTYContext)

	// ExitFULLTEXTSERVICEPROPERTY is called when exiting the FULLTEXTSERVICEPROPERTY production.
	ExitFULLTEXTSERVICEPROPERTY(c *FULLTEXTSERVICEPROPERTYContext)

	// ExitINDEX_COL is called when exiting the INDEX_COL production.
	ExitINDEX_COL(c *INDEX_COLContext)

	// ExitINDEXKEY_PROPERTY is called when exiting the INDEXKEY_PROPERTY production.
	ExitINDEXKEY_PROPERTY(c *INDEXKEY_PROPERTYContext)

	// ExitINDEXPROPERTY is called when exiting the INDEXPROPERTY production.
	ExitINDEXPROPERTY(c *INDEXPROPERTYContext)

	// ExitNEXT_VALUE_FOR is called when exiting the NEXT_VALUE_FOR production.
	ExitNEXT_VALUE_FOR(c *NEXT_VALUE_FORContext)

	// ExitOBJECT_DEFINITION is called when exiting the OBJECT_DEFINITION production.
	ExitOBJECT_DEFINITION(c *OBJECT_DEFINITIONContext)

	// ExitOBJECT_ID is called when exiting the OBJECT_ID production.
	ExitOBJECT_ID(c *OBJECT_IDContext)

	// ExitOBJECT_NAME is called when exiting the OBJECT_NAME production.
	ExitOBJECT_NAME(c *OBJECT_NAMEContext)

	// ExitOBJECT_SCHEMA_NAME is called when exiting the OBJECT_SCHEMA_NAME production.
	ExitOBJECT_SCHEMA_NAME(c *OBJECT_SCHEMA_NAMEContext)

	// ExitOBJECTPROPERTY is called when exiting the OBJECTPROPERTY production.
	ExitOBJECTPROPERTY(c *OBJECTPROPERTYContext)

	// ExitOBJECTPROPERTYEX is called when exiting the OBJECTPROPERTYEX production.
	ExitOBJECTPROPERTYEX(c *OBJECTPROPERTYEXContext)

	// ExitORIGINAL_DB_NAME is called when exiting the ORIGINAL_DB_NAME production.
	ExitORIGINAL_DB_NAME(c *ORIGINAL_DB_NAMEContext)

	// ExitPARSENAME is called when exiting the PARSENAME production.
	ExitPARSENAME(c *PARSENAMEContext)

	// ExitSCHEMA_ID is called when exiting the SCHEMA_ID production.
	ExitSCHEMA_ID(c *SCHEMA_IDContext)

	// ExitSCHEMA_NAME is called when exiting the SCHEMA_NAME production.
	ExitSCHEMA_NAME(c *SCHEMA_NAMEContext)

	// ExitSCOPE_IDENTITY is called when exiting the SCOPE_IDENTITY production.
	ExitSCOPE_IDENTITY(c *SCOPE_IDENTITYContext)

	// ExitSERVERPROPERTY is called when exiting the SERVERPROPERTY production.
	ExitSERVERPROPERTY(c *SERVERPROPERTYContext)

	// ExitSTATS_DATE is called when exiting the STATS_DATE production.
	ExitSTATS_DATE(c *STATS_DATEContext)

	// ExitTYPE_ID is called when exiting the TYPE_ID production.
	ExitTYPE_ID(c *TYPE_IDContext)

	// ExitTYPE_NAME is called when exiting the TYPE_NAME production.
	ExitTYPE_NAME(c *TYPE_NAMEContext)

	// ExitTYPEPROPERTY is called when exiting the TYPEPROPERTY production.
	ExitTYPEPROPERTY(c *TYPEPROPERTYContext)

	// ExitASCII is called when exiting the ASCII production.
	ExitASCII(c *ASCIIContext)

	// ExitCHAR is called when exiting the CHAR production.
	ExitCHAR(c *CHARContext)

	// ExitCHARINDEX is called when exiting the CHARINDEX production.
	ExitCHARINDEX(c *CHARINDEXContext)

	// ExitCONCAT is called when exiting the CONCAT production.
	ExitCONCAT(c *CONCATContext)

	// ExitCONCAT_WS is called when exiting the CONCAT_WS production.
	ExitCONCAT_WS(c *CONCAT_WSContext)

	// ExitDIFFERENCE is called when exiting the DIFFERENCE production.
	ExitDIFFERENCE(c *DIFFERENCEContext)

	// ExitFORMAT is called when exiting the FORMAT production.
	ExitFORMAT(c *FORMATContext)

	// ExitLEFT is called when exiting the LEFT production.
	ExitLEFT(c *LEFTContext)

	// ExitLEN is called when exiting the LEN production.
	ExitLEN(c *LENContext)

	// ExitLOWER is called when exiting the LOWER production.
	ExitLOWER(c *LOWERContext)

	// ExitLTRIM is called when exiting the LTRIM production.
	ExitLTRIM(c *LTRIMContext)

	// ExitNCHAR is called when exiting the NCHAR production.
	ExitNCHAR(c *NCHARContext)

	// ExitPATINDEX is called when exiting the PATINDEX production.
	ExitPATINDEX(c *PATINDEXContext)

	// ExitQUOTENAME is called when exiting the QUOTENAME production.
	ExitQUOTENAME(c *QUOTENAMEContext)

	// ExitREPLACE is called when exiting the REPLACE production.
	ExitREPLACE(c *REPLACEContext)

	// ExitREPLICATE is called when exiting the REPLICATE production.
	ExitREPLICATE(c *REPLICATEContext)

	// ExitREVERSE is called when exiting the REVERSE production.
	ExitREVERSE(c *REVERSEContext)

	// ExitRIGHT is called when exiting the RIGHT production.
	ExitRIGHT(c *RIGHTContext)

	// ExitRTRIM is called when exiting the RTRIM production.
	ExitRTRIM(c *RTRIMContext)

	// ExitSOUNDEX is called when exiting the SOUNDEX production.
	ExitSOUNDEX(c *SOUNDEXContext)

	// ExitSPACE is called when exiting the SPACE production.
	ExitSPACE(c *SPACEContext)

	// ExitSTR is called when exiting the STR production.
	ExitSTR(c *STRContext)

	// ExitSTRINGAGG is called when exiting the STRINGAGG production.
	ExitSTRINGAGG(c *STRINGAGGContext)

	// ExitSTRING_ESCAPE is called when exiting the STRING_ESCAPE production.
	ExitSTRING_ESCAPE(c *STRING_ESCAPEContext)

	// ExitSTUFF is called when exiting the STUFF production.
	ExitSTUFF(c *STUFFContext)

	// ExitSUBSTRING is called when exiting the SUBSTRING production.
	ExitSUBSTRING(c *SUBSTRINGContext)

	// ExitTRANSLATE is called when exiting the TRANSLATE production.
	ExitTRANSLATE(c *TRANSLATEContext)

	// ExitTRIM is called when exiting the TRIM production.
	ExitTRIM(c *TRIMContext)

	// ExitUNICODE is called when exiting the UNICODE production.
	ExitUNICODE(c *UNICODEContext)

	// ExitUPPER is called when exiting the UPPER production.
	ExitUPPER(c *UPPERContext)

	// ExitBINARY_CHECKSUM is called when exiting the BINARY_CHECKSUM production.
	ExitBINARY_CHECKSUM(c *BINARY_CHECKSUMContext)

	// ExitCHECKSUM is called when exiting the CHECKSUM production.
	ExitCHECKSUM(c *CHECKSUMContext)

	// ExitCOMPRESS is called when exiting the COMPRESS production.
	ExitCOMPRESS(c *COMPRESSContext)

	// ExitCONNECTIONPROPERTY is called when exiting the CONNECTIONPROPERTY production.
	ExitCONNECTIONPROPERTY(c *CONNECTIONPROPERTYContext)

	// ExitCONTEXT_INFO is called when exiting the CONTEXT_INFO production.
	ExitCONTEXT_INFO(c *CONTEXT_INFOContext)

	// ExitCURRENT_REQUEST_ID is called when exiting the CURRENT_REQUEST_ID production.
	ExitCURRENT_REQUEST_ID(c *CURRENT_REQUEST_IDContext)

	// ExitCURRENT_TRANSACTION_ID is called when exiting the CURRENT_TRANSACTION_ID production.
	ExitCURRENT_TRANSACTION_ID(c *CURRENT_TRANSACTION_IDContext)

	// ExitDECOMPRESS is called when exiting the DECOMPRESS production.
	ExitDECOMPRESS(c *DECOMPRESSContext)

	// ExitERROR_LINE is called when exiting the ERROR_LINE production.
	ExitERROR_LINE(c *ERROR_LINEContext)

	// ExitERROR_MESSAGE is called when exiting the ERROR_MESSAGE production.
	ExitERROR_MESSAGE(c *ERROR_MESSAGEContext)

	// ExitERROR_NUMBER is called when exiting the ERROR_NUMBER production.
	ExitERROR_NUMBER(c *ERROR_NUMBERContext)

	// ExitERROR_PROCEDURE is called when exiting the ERROR_PROCEDURE production.
	ExitERROR_PROCEDURE(c *ERROR_PROCEDUREContext)

	// ExitERROR_SEVERITY is called when exiting the ERROR_SEVERITY production.
	ExitERROR_SEVERITY(c *ERROR_SEVERITYContext)

	// ExitERROR_STATE is called when exiting the ERROR_STATE production.
	ExitERROR_STATE(c *ERROR_STATEContext)

	// ExitFORMATMESSAGE is called when exiting the FORMATMESSAGE production.
	ExitFORMATMESSAGE(c *FORMATMESSAGEContext)

	// ExitGET_FILESTREAM_TRANSACTION_CONTEXT is called when exiting the GET_FILESTREAM_TRANSACTION_CONTEXT production.
	ExitGET_FILESTREAM_TRANSACTION_CONTEXT(c *GET_FILESTREAM_TRANSACTION_CONTEXTContext)

	// ExitGETANSINULL is called when exiting the GETANSINULL production.
	ExitGETANSINULL(c *GETANSINULLContext)

	// ExitHOST_ID is called when exiting the HOST_ID production.
	ExitHOST_ID(c *HOST_IDContext)

	// ExitHOST_NAME is called when exiting the HOST_NAME production.
	ExitHOST_NAME(c *HOST_NAMEContext)

	// ExitISNULL is called when exiting the ISNULL production.
	ExitISNULL(c *ISNULLContext)

	// ExitISNUMERIC is called when exiting the ISNUMERIC production.
	ExitISNUMERIC(c *ISNUMERICContext)

	// ExitMIN_ACTIVE_ROWVERSION is called when exiting the MIN_ACTIVE_ROWVERSION production.
	ExitMIN_ACTIVE_ROWVERSION(c *MIN_ACTIVE_ROWVERSIONContext)

	// ExitNEWID is called when exiting the NEWID production.
	ExitNEWID(c *NEWIDContext)

	// ExitNEWSEQUENTIALID is called when exiting the NEWSEQUENTIALID production.
	ExitNEWSEQUENTIALID(c *NEWSEQUENTIALIDContext)

	// ExitROWCOUNT_BIG is called when exiting the ROWCOUNT_BIG production.
	ExitROWCOUNT_BIG(c *ROWCOUNT_BIGContext)

	// ExitSESSION_CONTEXT is called when exiting the SESSION_CONTEXT production.
	ExitSESSION_CONTEXT(c *SESSION_CONTEXTContext)

	// ExitXACT_STATE is called when exiting the XACT_STATE production.
	ExitXACT_STATE(c *XACT_STATEContext)

	// ExitCAST is called when exiting the CAST production.
	ExitCAST(c *CASTContext)

	// ExitTRY_CAST is called when exiting the TRY_CAST production.
	ExitTRY_CAST(c *TRY_CASTContext)

	// ExitCONVERT is called when exiting the CONVERT production.
	ExitCONVERT(c *CONVERTContext)

	// ExitCOALESCE is called when exiting the COALESCE production.
	ExitCOALESCE(c *COALESCEContext)

	// ExitCURSOR_ROWS is called when exiting the CURSOR_ROWS production.
	ExitCURSOR_ROWS(c *CURSOR_ROWSContext)

	// ExitFETCH_STATUS is called when exiting the FETCH_STATUS production.
	ExitFETCH_STATUS(c *FETCH_STATUSContext)

	// ExitCURSOR_STATUS is called when exiting the CURSOR_STATUS production.
	ExitCURSOR_STATUS(c *CURSOR_STATUSContext)

	// ExitCERT_ID is called when exiting the CERT_ID production.
	ExitCERT_ID(c *CERT_IDContext)

	// ExitDATALENGTH is called when exiting the DATALENGTH production.
	ExitDATALENGTH(c *DATALENGTHContext)

	// ExitIDENT_CURRENT is called when exiting the IDENT_CURRENT production.
	ExitIDENT_CURRENT(c *IDENT_CURRENTContext)

	// ExitIDENT_INCR is called when exiting the IDENT_INCR production.
	ExitIDENT_INCR(c *IDENT_INCRContext)

	// ExitIDENT_SEED is called when exiting the IDENT_SEED production.
	ExitIDENT_SEED(c *IDENT_SEEDContext)

	// ExitIDENTITY is called when exiting the IDENTITY production.
	ExitIDENTITY(c *IDENTITYContext)

	// ExitSQL_VARIANT_PROPERTY is called when exiting the SQL_VARIANT_PROPERTY production.
	ExitSQL_VARIANT_PROPERTY(c *SQL_VARIANT_PROPERTYContext)

	// ExitCURRENT_DATE is called when exiting the CURRENT_DATE production.
	ExitCURRENT_DATE(c *CURRENT_DATEContext)

	// ExitCURRENT_TIMESTAMP is called when exiting the CURRENT_TIMESTAMP production.
	ExitCURRENT_TIMESTAMP(c *CURRENT_TIMESTAMPContext)

	// ExitCURRENT_TIMEZONE is called when exiting the CURRENT_TIMEZONE production.
	ExitCURRENT_TIMEZONE(c *CURRENT_TIMEZONEContext)

	// ExitCURRENT_TIMEZONE_ID is called when exiting the CURRENT_TIMEZONE_ID production.
	ExitCURRENT_TIMEZONE_ID(c *CURRENT_TIMEZONE_IDContext)

	// ExitDATE_BUCKET is called when exiting the DATE_BUCKET production.
	ExitDATE_BUCKET(c *DATE_BUCKETContext)

	// ExitDATEADD is called when exiting the DATEADD production.
	ExitDATEADD(c *DATEADDContext)

	// ExitDATEDIFF is called when exiting the DATEDIFF production.
	ExitDATEDIFF(c *DATEDIFFContext)

	// ExitDATEDIFF_BIG is called when exiting the DATEDIFF_BIG production.
	ExitDATEDIFF_BIG(c *DATEDIFF_BIGContext)

	// ExitDATEFROMPARTS is called when exiting the DATEFROMPARTS production.
	ExitDATEFROMPARTS(c *DATEFROMPARTSContext)

	// ExitDATENAME is called when exiting the DATENAME production.
	ExitDATENAME(c *DATENAMEContext)

	// ExitDATEPART is called when exiting the DATEPART production.
	ExitDATEPART(c *DATEPARTContext)

	// ExitDATETIME2FROMPARTS is called when exiting the DATETIME2FROMPARTS production.
	ExitDATETIME2FROMPARTS(c *DATETIME2FROMPARTSContext)

	// ExitDATETIMEFROMPARTS is called when exiting the DATETIMEFROMPARTS production.
	ExitDATETIMEFROMPARTS(c *DATETIMEFROMPARTSContext)

	// ExitDATETIMEOFFSETFROMPARTS is called when exiting the DATETIMEOFFSETFROMPARTS production.
	ExitDATETIMEOFFSETFROMPARTS(c *DATETIMEOFFSETFROMPARTSContext)

	// ExitDATETRUNC is called when exiting the DATETRUNC production.
	ExitDATETRUNC(c *DATETRUNCContext)

	// ExitDAY is called when exiting the DAY production.
	ExitDAY(c *DAYContext)

	// ExitEOMONTH is called when exiting the EOMONTH production.
	ExitEOMONTH(c *EOMONTHContext)

	// ExitGETDATE is called when exiting the GETDATE production.
	ExitGETDATE(c *GETDATEContext)

	// ExitGETUTCDATE is called when exiting the GETUTCDATE production.
	ExitGETUTCDATE(c *GETUTCDATEContext)

	// ExitISDATE is called when exiting the ISDATE production.
	ExitISDATE(c *ISDATEContext)

	// ExitMONTH is called when exiting the MONTH production.
	ExitMONTH(c *MONTHContext)

	// ExitSMALLDATETIMEFROMPARTS is called when exiting the SMALLDATETIMEFROMPARTS production.
	ExitSMALLDATETIMEFROMPARTS(c *SMALLDATETIMEFROMPARTSContext)

	// ExitSWITCHOFFSET is called when exiting the SWITCHOFFSET production.
	ExitSWITCHOFFSET(c *SWITCHOFFSETContext)

	// ExitSYSDATETIME is called when exiting the SYSDATETIME production.
	ExitSYSDATETIME(c *SYSDATETIMEContext)

	// ExitSYSDATETIMEOFFSET is called when exiting the SYSDATETIMEOFFSET production.
	ExitSYSDATETIMEOFFSET(c *SYSDATETIMEOFFSETContext)

	// ExitSYSUTCDATETIME is called when exiting the SYSUTCDATETIME production.
	ExitSYSUTCDATETIME(c *SYSUTCDATETIMEContext)

	// ExitTIMEFROMPARTS is called when exiting the TIMEFROMPARTS production.
	ExitTIMEFROMPARTS(c *TIMEFROMPARTSContext)

	// ExitTODATETIMEOFFSET is called when exiting the TODATETIMEOFFSET production.
	ExitTODATETIMEOFFSET(c *TODATETIMEOFFSETContext)

	// ExitYEAR is called when exiting the YEAR production.
	ExitYEAR(c *YEARContext)

	// ExitNULLIF is called when exiting the NULLIF production.
	ExitNULLIF(c *NULLIFContext)

	// ExitPARSE is called when exiting the PARSE production.
	ExitPARSE(c *PARSEContext)

	// ExitXML_DATA_TYPE_FUNC is called when exiting the XML_DATA_TYPE_FUNC production.
	ExitXML_DATA_TYPE_FUNC(c *XML_DATA_TYPE_FUNCContext)

	// ExitIIF is called when exiting the IIF production.
	ExitIIF(c *IIFContext)

	// ExitISJSON is called when exiting the ISJSON production.
	ExitISJSON(c *ISJSONContext)

	// ExitJSON_OBJECT is called when exiting the JSON_OBJECT production.
	ExitJSON_OBJECT(c *JSON_OBJECTContext)

	// ExitJSON_ARRAY is called when exiting the JSON_ARRAY production.
	ExitJSON_ARRAY(c *JSON_ARRAYContext)

	// ExitJSON_VALUE is called when exiting the JSON_VALUE production.
	ExitJSON_VALUE(c *JSON_VALUEContext)

	// ExitJSON_QUERY is called when exiting the JSON_QUERY production.
	ExitJSON_QUERY(c *JSON_QUERYContext)

	// ExitJSON_MODIFY is called when exiting the JSON_MODIFY production.
	ExitJSON_MODIFY(c *JSON_MODIFYContext)

	// ExitJSON_PATH_EXISTS is called when exiting the JSON_PATH_EXISTS production.
	ExitJSON_PATH_EXISTS(c *JSON_PATH_EXISTSContext)

	// ExitABS is called when exiting the ABS production.
	ExitABS(c *ABSContext)

	// ExitACOS is called when exiting the ACOS production.
	ExitACOS(c *ACOSContext)

	// ExitASIN is called when exiting the ASIN production.
	ExitASIN(c *ASINContext)

	// ExitATAN is called when exiting the ATAN production.
	ExitATAN(c *ATANContext)

	// ExitATN2 is called when exiting the ATN2 production.
	ExitATN2(c *ATN2Context)

	// ExitCEILING is called when exiting the CEILING production.
	ExitCEILING(c *CEILINGContext)

	// ExitCOS is called when exiting the COS production.
	ExitCOS(c *COSContext)

	// ExitCOT is called when exiting the COT production.
	ExitCOT(c *COTContext)

	// ExitDEGREES is called when exiting the DEGREES production.
	ExitDEGREES(c *DEGREESContext)

	// ExitEXP is called when exiting the EXP production.
	ExitEXP(c *EXPContext)

	// ExitFLOOR is called when exiting the FLOOR production.
	ExitFLOOR(c *FLOORContext)

	// ExitLOG is called when exiting the LOG production.
	ExitLOG(c *LOGContext)

	// ExitLOG10 is called when exiting the LOG10 production.
	ExitLOG10(c *LOG10Context)

	// ExitPI is called when exiting the PI production.
	ExitPI(c *PIContext)

	// ExitPOWER is called when exiting the POWER production.
	ExitPOWER(c *POWERContext)

	// ExitRADIANS is called when exiting the RADIANS production.
	ExitRADIANS(c *RADIANSContext)

	// ExitRAND is called when exiting the RAND production.
	ExitRAND(c *RANDContext)

	// ExitROUND is called when exiting the ROUND production.
	ExitROUND(c *ROUNDContext)

	// ExitMATH_SIGN is called when exiting the MATH_SIGN production.
	ExitMATH_SIGN(c *MATH_SIGNContext)

	// ExitSIN is called when exiting the SIN production.
	ExitSIN(c *SINContext)

	// ExitSQRT is called when exiting the SQRT production.
	ExitSQRT(c *SQRTContext)

	// ExitSQUARE is called when exiting the SQUARE production.
	ExitSQUARE(c *SQUAREContext)

	// ExitTAN is called when exiting the TAN production.
	ExitTAN(c *TANContext)

	// ExitGREATEST is called when exiting the GREATEST production.
	ExitGREATEST(c *GREATESTContext)

	// ExitLEAST is called when exiting the LEAST production.
	ExitLEAST(c *LEASTContext)

	// ExitCERTENCODED is called when exiting the CERTENCODED production.
	ExitCERTENCODED(c *CERTENCODEDContext)

	// ExitCERTPRIVATEKEY is called when exiting the CERTPRIVATEKEY production.
	ExitCERTPRIVATEKEY(c *CERTPRIVATEKEYContext)

	// ExitCURRENT_USER is called when exiting the CURRENT_USER production.
	ExitCURRENT_USER(c *CURRENT_USERContext)

	// ExitDATABASE_PRINCIPAL_ID is called when exiting the DATABASE_PRINCIPAL_ID production.
	ExitDATABASE_PRINCIPAL_ID(c *DATABASE_PRINCIPAL_IDContext)

	// ExitHAS_DBACCESS is called when exiting the HAS_DBACCESS production.
	ExitHAS_DBACCESS(c *HAS_DBACCESSContext)

	// ExitHAS_PERMS_BY_NAME is called when exiting the HAS_PERMS_BY_NAME production.
	ExitHAS_PERMS_BY_NAME(c *HAS_PERMS_BY_NAMEContext)

	// ExitIS_MEMBER is called when exiting the IS_MEMBER production.
	ExitIS_MEMBER(c *IS_MEMBERContext)

	// ExitIS_ROLEMEMBER is called when exiting the IS_ROLEMEMBER production.
	ExitIS_ROLEMEMBER(c *IS_ROLEMEMBERContext)

	// ExitIS_SRVROLEMEMBER is called when exiting the IS_SRVROLEMEMBER production.
	ExitIS_SRVROLEMEMBER(c *IS_SRVROLEMEMBERContext)

	// ExitLOGINPROPERTY is called when exiting the LOGINPROPERTY production.
	ExitLOGINPROPERTY(c *LOGINPROPERTYContext)

	// ExitORIGINAL_LOGIN is called when exiting the ORIGINAL_LOGIN production.
	ExitORIGINAL_LOGIN(c *ORIGINAL_LOGINContext)

	// ExitPERMISSIONS is called when exiting the PERMISSIONS production.
	ExitPERMISSIONS(c *PERMISSIONSContext)

	// ExitPWDENCRYPT is called when exiting the PWDENCRYPT production.
	ExitPWDENCRYPT(c *PWDENCRYPTContext)

	// ExitPWDCOMPARE is called when exiting the PWDCOMPARE production.
	ExitPWDCOMPARE(c *PWDCOMPAREContext)

	// ExitSESSION_USER is called when exiting the SESSION_USER production.
	ExitSESSION_USER(c *SESSION_USERContext)

	// ExitSESSIONPROPERTY is called when exiting the SESSIONPROPERTY production.
	ExitSESSIONPROPERTY(c *SESSIONPROPERTYContext)

	// ExitSUSER_ID is called when exiting the SUSER_ID production.
	ExitSUSER_ID(c *SUSER_IDContext)

	// ExitSUSER_SNAME is called when exiting the SUSER_SNAME production.
	ExitSUSER_SNAME(c *SUSER_SNAMEContext)

	// ExitSUSER_SID is called when exiting the SUSER_SID production.
	ExitSUSER_SID(c *SUSER_SIDContext)

	// ExitSYSTEM_USER is called when exiting the SYSTEM_USER production.
	ExitSYSTEM_USER(c *SYSTEM_USERContext)

	// ExitUSER is called when exiting the USER production.
	ExitUSER(c *USERContext)

	// ExitUSER_ID is called when exiting the USER_ID production.
	ExitUSER_ID(c *USER_IDContext)

	// ExitUSER_NAME is called when exiting the USER_NAME production.
	ExitUSER_NAME(c *USER_NAMEContext)

	// ExitXml_data_type_methods is called when exiting the xml_data_type_methods production.
	ExitXml_data_type_methods(c *Xml_data_type_methodsContext)

	// ExitDateparts_9 is called when exiting the dateparts_9 production.
	ExitDateparts_9(c *Dateparts_9Context)

	// ExitDateparts_12 is called when exiting the dateparts_12 production.
	ExitDateparts_12(c *Dateparts_12Context)

	// ExitDateparts_15 is called when exiting the dateparts_15 production.
	ExitDateparts_15(c *Dateparts_15Context)

	// ExitDateparts_datetrunc is called when exiting the dateparts_datetrunc production.
	ExitDateparts_datetrunc(c *Dateparts_datetruncContext)

	// ExitValue_method is called when exiting the value_method production.
	ExitValue_method(c *Value_methodContext)

	// ExitValue_call is called when exiting the value_call production.
	ExitValue_call(c *Value_callContext)

	// ExitQuery_method is called when exiting the query_method production.
	ExitQuery_method(c *Query_methodContext)

	// ExitQuery_call is called when exiting the query_call production.
	ExitQuery_call(c *Query_callContext)

	// ExitExist_method is called when exiting the exist_method production.
	ExitExist_method(c *Exist_methodContext)

	// ExitExist_call is called when exiting the exist_call production.
	ExitExist_call(c *Exist_callContext)

	// ExitModify_method is called when exiting the modify_method production.
	ExitModify_method(c *Modify_methodContext)

	// ExitModify_call is called when exiting the modify_call production.
	ExitModify_call(c *Modify_callContext)

	// ExitHierarchyid_call is called when exiting the hierarchyid_call production.
	ExitHierarchyid_call(c *Hierarchyid_callContext)

	// ExitHierarchyid_static_method is called when exiting the hierarchyid_static_method production.
	ExitHierarchyid_static_method(c *Hierarchyid_static_methodContext)

	// ExitNodes_method is called when exiting the nodes_method production.
	ExitNodes_method(c *Nodes_methodContext)

	// ExitSwitch_section is called when exiting the switch_section production.
	ExitSwitch_section(c *Switch_sectionContext)

	// ExitSwitch_search_condition_section is called when exiting the switch_search_condition_section production.
	ExitSwitch_search_condition_section(c *Switch_search_condition_sectionContext)

	// ExitAs_column_alias is called when exiting the as_column_alias production.
	ExitAs_column_alias(c *As_column_aliasContext)

	// ExitAs_table_alias is called when exiting the as_table_alias production.
	ExitAs_table_alias(c *As_table_aliasContext)

	// ExitTable_alias is called when exiting the table_alias production.
	ExitTable_alias(c *Table_aliasContext)

	// ExitWith_table_hints is called when exiting the with_table_hints production.
	ExitWith_table_hints(c *With_table_hintsContext)

	// ExitDeprecated_table_hint is called when exiting the deprecated_table_hint production.
	ExitDeprecated_table_hint(c *Deprecated_table_hintContext)

	// ExitSybase_legacy_hints is called when exiting the sybase_legacy_hints production.
	ExitSybase_legacy_hints(c *Sybase_legacy_hintsContext)

	// ExitSybase_legacy_hint is called when exiting the sybase_legacy_hint production.
	ExitSybase_legacy_hint(c *Sybase_legacy_hintContext)

	// ExitTable_hint is called when exiting the table_hint production.
	ExitTable_hint(c *Table_hintContext)

	// ExitIndex_value is called when exiting the index_value production.
	ExitIndex_value(c *Index_valueContext)

	// ExitColumn_alias_list is called when exiting the column_alias_list production.
	ExitColumn_alias_list(c *Column_alias_listContext)

	// ExitColumn_alias is called when exiting the column_alias production.
	ExitColumn_alias(c *Column_aliasContext)

	// ExitTable_value_constructor is called when exiting the table_value_constructor production.
	ExitTable_value_constructor(c *Table_value_constructorContext)

	// ExitExpression_list_ is called when exiting the expression_list_ production.
	ExitExpression_list_(c *Expression_list_Context)

	// ExitRanking_windowed_function is called when exiting the ranking_windowed_function production.
	ExitRanking_windowed_function(c *Ranking_windowed_functionContext)

	// ExitAggregate_windowed_function is called when exiting the aggregate_windowed_function production.
	ExitAggregate_windowed_function(c *Aggregate_windowed_functionContext)

	// ExitAnalytic_windowed_function is called when exiting the analytic_windowed_function production.
	ExitAnalytic_windowed_function(c *Analytic_windowed_functionContext)

	// ExitAll_distinct_expression is called when exiting the all_distinct_expression production.
	ExitAll_distinct_expression(c *All_distinct_expressionContext)

	// ExitOver_clause is called when exiting the over_clause production.
	ExitOver_clause(c *Over_clauseContext)

	// ExitRow_or_range_clause is called when exiting the row_or_range_clause production.
	ExitRow_or_range_clause(c *Row_or_range_clauseContext)

	// ExitWindow_frame_extent is called when exiting the window_frame_extent production.
	ExitWindow_frame_extent(c *Window_frame_extentContext)

	// ExitWindow_frame_bound is called when exiting the window_frame_bound production.
	ExitWindow_frame_bound(c *Window_frame_boundContext)

	// ExitWindow_frame_preceding is called when exiting the window_frame_preceding production.
	ExitWindow_frame_preceding(c *Window_frame_precedingContext)

	// ExitWindow_frame_following is called when exiting the window_frame_following production.
	ExitWindow_frame_following(c *Window_frame_followingContext)

	// ExitCreate_database_option is called when exiting the create_database_option production.
	ExitCreate_database_option(c *Create_database_optionContext)

	// ExitDatabase_filestream_option is called when exiting the database_filestream_option production.
	ExitDatabase_filestream_option(c *Database_filestream_optionContext)

	// ExitDatabase_file_spec is called when exiting the database_file_spec production.
	ExitDatabase_file_spec(c *Database_file_specContext)

	// ExitFile_group is called when exiting the file_group production.
	ExitFile_group(c *File_groupContext)

	// ExitFile_spec is called when exiting the file_spec production.
	ExitFile_spec(c *File_specContext)

	// ExitEntity_name is called when exiting the entity_name production.
	ExitEntity_name(c *Entity_nameContext)

	// ExitEntity_name_for_azure_dw is called when exiting the entity_name_for_azure_dw production.
	ExitEntity_name_for_azure_dw(c *Entity_name_for_azure_dwContext)

	// ExitEntity_name_for_parallel_dw is called when exiting the entity_name_for_parallel_dw production.
	ExitEntity_name_for_parallel_dw(c *Entity_name_for_parallel_dwContext)

	// ExitFull_table_name is called when exiting the full_table_name production.
	ExitFull_table_name(c *Full_table_nameContext)

	// ExitTable_name is called when exiting the table_name production.
	ExitTable_name(c *Table_nameContext)

	// ExitSimple_name is called when exiting the simple_name production.
	ExitSimple_name(c *Simple_nameContext)

	// ExitFunc_proc_name_schema is called when exiting the func_proc_name_schema production.
	ExitFunc_proc_name_schema(c *Func_proc_name_schemaContext)

	// ExitFunc_proc_name_database_schema is called when exiting the func_proc_name_database_schema production.
	ExitFunc_proc_name_database_schema(c *Func_proc_name_database_schemaContext)

	// ExitFunc_proc_name_server_database_schema is called when exiting the func_proc_name_server_database_schema production.
	ExitFunc_proc_name_server_database_schema(c *Func_proc_name_server_database_schemaContext)

	// ExitDdl_object is called when exiting the ddl_object production.
	ExitDdl_object(c *Ddl_objectContext)

	// ExitFull_column_name is called when exiting the full_column_name production.
	ExitFull_column_name(c *Full_column_nameContext)

	// ExitColumn_name_list_with_order is called when exiting the column_name_list_with_order production.
	ExitColumn_name_list_with_order(c *Column_name_list_with_orderContext)

	// ExitInsert_column_name_list is called when exiting the insert_column_name_list production.
	ExitInsert_column_name_list(c *Insert_column_name_listContext)

	// ExitInsert_column_id is called when exiting the insert_column_id production.
	ExitInsert_column_id(c *Insert_column_idContext)

	// ExitColumn_name_list is called when exiting the column_name_list production.
	ExitColumn_name_list(c *Column_name_listContext)

	// ExitCursor_name is called when exiting the cursor_name production.
	ExitCursor_name(c *Cursor_nameContext)

	// ExitOn_off is called when exiting the on_off production.
	ExitOn_off(c *On_offContext)

	// ExitClustered is called when exiting the clustered production.
	ExitClustered(c *ClusteredContext)

	// ExitNull_notnull is called when exiting the null_notnull production.
	ExitNull_notnull(c *Null_notnullContext)

	// ExitScalar_function_name is called when exiting the scalar_function_name production.
	ExitScalar_function_name(c *Scalar_function_nameContext)

	// ExitBegin_conversation_timer is called when exiting the begin_conversation_timer production.
	ExitBegin_conversation_timer(c *Begin_conversation_timerContext)

	// ExitBegin_conversation_dialog is called when exiting the begin_conversation_dialog production.
	ExitBegin_conversation_dialog(c *Begin_conversation_dialogContext)

	// ExitContract_name is called when exiting the contract_name production.
	ExitContract_name(c *Contract_nameContext)

	// ExitService_name is called when exiting the service_name production.
	ExitService_name(c *Service_nameContext)

	// ExitEnd_conversation is called when exiting the end_conversation production.
	ExitEnd_conversation(c *End_conversationContext)

	// ExitWaitfor_conversation is called when exiting the waitfor_conversation production.
	ExitWaitfor_conversation(c *Waitfor_conversationContext)

	// ExitGet_conversation is called when exiting the get_conversation production.
	ExitGet_conversation(c *Get_conversationContext)

	// ExitQueue_id is called when exiting the queue_id production.
	ExitQueue_id(c *Queue_idContext)

	// ExitSend_conversation is called when exiting the send_conversation production.
	ExitSend_conversation(c *Send_conversationContext)

	// ExitData_type is called when exiting the data_type production.
	ExitData_type(c *Data_typeContext)

	// ExitConstant is called when exiting the constant production.
	ExitConstant(c *ConstantContext)

	// ExitPrimitive_constant is called when exiting the primitive_constant production.
	ExitPrimitive_constant(c *Primitive_constantContext)

	// ExitKeyword is called when exiting the keyword production.
	ExitKeyword(c *KeywordContext)

	// ExitId_ is called when exiting the id_ production.
	ExitId_(c *Id_Context)

	// ExitSimple_id is called when exiting the simple_id production.
	ExitSimple_id(c *Simple_idContext)

	// ExitId_or_string is called when exiting the id_or_string production.
	ExitId_or_string(c *Id_or_stringContext)

	// ExitComparison_operator is called when exiting the comparison_operator production.
	ExitComparison_operator(c *Comparison_operatorContext)

	// ExitAssignment_operator is called when exiting the assignment_operator production.
	ExitAssignment_operator(c *Assignment_operatorContext)

	// ExitFile_size is called when exiting the file_size production.
	ExitFile_size(c *File_sizeContext)
}
