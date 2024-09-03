import React from "react";
import Layout from "@theme/Layout";
import { DatabasePerTenant } from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <Layout title="Schema Migrations for Database per Tenant with Atlas">
      <DatabasePerTenant />
    </Layout>
  );
}
