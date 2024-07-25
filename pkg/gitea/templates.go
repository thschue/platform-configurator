package gitea

type AppSetTemplateData struct {
	Stage       string
	GiteaSshUrl string
	GitOrg      string
	GitRepo     string
	ArgoProject string
	ArgoCluster string
}

const appSetTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: platform-{{ .Stage }}
spec:
  generators:
    - git:
        repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        revision: main
        files:
          - path: {{ .Stage }}/**/config.yaml
  template:
    metadata:
      name: {{ "'{{path.basename}}'" }}
    spec:
      project: {{ .ArgoProject }}
      sources:
      - repoURL: {{ "'{{repoURL}}'" }}
        targetRevision: {{ "'{{targetRevision}}'" }}
        chart: {{ "'{{chart}}'" }}
        helm:
          valueFiles:
          - $values/{{ .Stage }}/{{ "{{path.basename}}" }}/values.yaml
      - repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        targetRevision: main
        path: {{ .Stage }}/{{ "{{path.basename}}" }}
      - repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        targetRevision: main
        ref: values
      destination:
        name: {{ .ArgoCluster }}
        namespace: {{ "'{{path.basename}}'" }}
      syncPolicy:
        automated: 
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
          - ServerSideApply={{ "'{{serverSideApply}}'" }}
`
