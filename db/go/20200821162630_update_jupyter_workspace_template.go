package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	uid2 "github.com/onepanelio/core/pkg/util/uid"
	"github.com/pressly/goose"
)

const jupyterWorkspaceTemplate2 = `# Docker containers that are part of the Workspace
containers:
- name: jupyterlab-tensorflow
  image: onepanel/jupyterlab:1.0.1
  command: ["/bin/bash", "-c", "start.sh jupyter lab --LabApp.token='' --LabApp.allow_remote_access=True --LabApp.allow_origin=\"*\" --LabApp.disable_check_xsrf=True --LabApp.trust_xheaders=True --LabApp.base_url=/ --LabApp.tornado_settings='{\"headers\":{\"Content-Security-Policy\":\"frame-ancestors * \'self\'\"}}' --notebook-dir='/data' --allow-root"]
  env:
    - name: tornado
      value: "'{'headers':{'Content-Security-Policy':\"frame-ancestors\ *\ \'self'\"}}'"
  args:
  ports:
  - containerPort: 8888
    name: jupyterlab
  - containerPort: 6006
    name: tensorboard
  volumeMounts:
  - name: data
    mountPath: /data
ports:
- name: jupyterlab
  port: 80
  protocol: TCP
  targetPort: 8888
- name: tensorboard
  port: 6006
  protocol: TCP
  targetPort: 6006
routes:
- match:
  - uri:
      prefix: /tensorboard
  route:
  - destination:
      port:
        number: 6006
- match:
  - uri:
      prefix: / #jupyter runs at the default route
  route:
  - destination:
      port:
        number: 80
# DAG Workflow to be executed once a Workspace action completes (optional)        
#postExecutionWorkflow:
#  entrypoint: main
#  templates:
#  - name: main
#    dag:
#       tasks:
#       - name: slack-notify
#         template: slack-notify
#  - name: slack-notify
#    container:
#      image: technosophos/slack-notify
#      args:
#      - SLACK_USERNAME=onepanel SLACK_TITLE="Your workspace is ready" SLACK_ICON=https://www.gravatar.com/avatar/5c4478592fe00878f62f0027be59c1bd SLACK_MESSAGE="Your workspace is now running" ./slack-notify
#      command:
#      - sh
#      - -c
`

func initialize20200821162630() {
	if _, ok := initializedMigrations[20200821162630]; !ok {
		goose.AddMigration(Up20200821162630, Down20200821162630)
		initializedMigrations[20200821162630] = true
	}
}

//Up20200821162630 updates jupyterlab workspace with new features.
// Tensorflow 2.3 compiled to support CUDA 10.2 and cudnn8
// Pytorch with GPU support.
// OpenCV compiled from source with CUDA support.
// Miniconda, Jupyterlab 2.2+, and lots of neat extensions.
func Up20200821162630(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200821162630]; ok {
		return nil
	}

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(jupyterLabTemplateName, 30)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		workspaceTemplate := &v1.WorkspaceTemplate{
			UID:      uid,
			Name:     jupyterLabTemplateName,
			Manifest: jupyterWorkspaceTemplate2,
		}

		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}

// Down20200821162630 removes the JupyterLab template update
func Down20200821162630(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.DB.Close()

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return err
	}

	uid, err := uid2.GenerateUID(jupyterLabTemplateName, 30)
	if err != nil {
		return err
	}
	workspaceTemplate := &v1.WorkspaceTemplate{
		UID:      uid,
		Name:     jupyterLabTemplateName,
		Manifest: jupyterWorkspaceTemplate,
	}

	for _, namespace := range namespaces {
		if _, err := client.UpdateWorkspaceTemplate(namespace.Name, workspaceTemplate); err != nil {
			return err
		}
	}

	return nil
}
