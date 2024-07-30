module.exports = {
  links: [
    {
      title: "Docs",
      items: [
        { label: "Home", to: "docs" },
        { label: "Atlas vs Others ", to: "atlas-vs-others" },
        { label: "CLI Reference", to: "cli-reference" },
        { label: "Blog", to: "blog" },
        { label: "Guides", to: "guides" },
        { label: "GoDoc", to: "https://pkg.go.dev/ariga.io/atlas" },
      ],
    },
    {
      title: "Community",
      items: [
        { label: "GitHub", to: "https://github.com/ariga/atlas" },
        { label: "Discord", to: "https://discord.gg/zZ6sWVg6NT" },
        { label: "Twitter", to: "https://twitter.com/atlasgo_io" },
        { label: "Newsletter", to: "https://atlasnewsletter.substack.com/" },
        { label: "YouTube", to: "https://youtube.com/@ariga_io" },
      ],
    },
    {
      title: "Integrations",
      items: [
        { label: "GitHub Actions", to: "/integrations/github-actions" },
        { label: "Kubernetes Operator", to: "/integrations/kubernetes/operator" },
        { label: "Terraform", to: "/integrations/terraform-provider" },
        { label: "Go API", to: "/integrations/go-api" },
      ],
    },
    {
      title: "Use Cases",
      items: [
        { label: "Database per Tenant", to: "/use-cases/database-per-tenant" },
        { label: "Atlas Cloud", to: "https://atlasgo.cloud/?utm_term=footer" },
        { label: "Modern Database CI/CD", to: "https://atlasgo.io/guides/modern-database-ci-cd" },

      ],
    },
    {
      title: "Legal",
      items: [
        { label: "Privacy Policy", to: "https://ariga.io/legal/privacy" },
        { label: "Terms of Service", to: "https://ariga.io/legal/tos" },
      ],
    },
  ],
  copyright: `
    Copyright © ${new Date().getFullYear()} The Atlas Authors.
    The Go gopher was designed by <a href="http://reneefrench.blogspot.com/">Renee French</a>.
    <br/>
    The design for the Go gopher is licensed under the Creative Commons 3.0 Attributions license. Read this 
    <a href="https://blog.golang.org/gopher">article</a> for more details.
    <br/>
    `,
};
