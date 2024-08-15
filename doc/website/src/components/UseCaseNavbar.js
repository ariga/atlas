import React from "react";
import Tenant from "../assets/icons/use-cases/tenant.svg";
import Migrations from "../assets/icons/use-cases/migrations.svg";
import Monitoring from "../assets/icons/use-cases/monitoring.svg";
import Governance from "../assets/icons/use-cases/governance.svg";

import "./UseCaseNavbar.css";

const UseCasesMap = {
  migrations: {
    Icon: Migrations,
    title: "Modernize Schema Migrations",
    text: "Apply modern CI/CD to schema changes",
    href: "/use-cases/modernize-database-migrations",
  },
  monitoring: {
    Icon: Monitoring,
    title: "Standardize Schema Migrations",
    text: "One migration tool to rule them all",
    href: "/use-cases/standardize-schema-migrations",
  },
  tenant: {
    Icon: Tenant,
    title: "Database per tenant",
    text: "Manage thousands of databases as one",
    href: "/use-cases/database-per-tenant",
  },
  governance: {
    Icon: Governance,
    title: "Database Governance",
    text: "End-to-end control and compliance",
    href: "",
  },
};

export const UseCaseNavbar = ({ content }) => {
  const selectedCase = UseCasesMap[content];

  if (!selectedCase) {
    return null;
  }
  return (
    <a href={selectedCase.href || '#'} className={`use-case-link ${selectedCase.href ? '' : 'disabled'}`}>
      <selectedCase.Icon />
      <div>
        <span className="use-case-link-title">
          {selectedCase.title}
          {!selectedCase.href && <div>Coming soon</div>}
        </span>
        <span>{selectedCase.text}</span>
      </div>
    </a>
  );
};
