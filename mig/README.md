mig
===

Package mig provides tools to version, record and migrate a project schema, and also provides rules
to migrate the project data.

Project dom nodes are assigned a semantic `Version`, that are automatically determined based on the
node's content and its last known version. The version starts at v0.0.0 for new nodes and increments
either the minor or the patch version if the old and new definition differ.

The distinction between minor and patch version is made by considering the impact of the changes. As
a rule of thumb we want the minor version to increment if the change requires a db schema migration.
User should be able to explicitly bump major and minor versions.

A project `Manifest` contains the version information for the project and all of its nodes. Each
version includes two sha256 hashes for the node contents, seperated by minor and patch version
relevant data.

All datasets like backups or databases should store project versions. Programs involved with data
migration have the full project history to calculate any new versions, other programs only need the
project manifest.

The schema history and manifest are managed by the daql command and are written to files. Changes
need to be explicitly recorded into the project history and manifest. Data migration rules should
also be recorded for each version as part of the history. Simple migration rules can be expressed
as xelf expressions and interpreted by the daql command. Complex migration rules call any command,
usually a simple go script that migrates one or more model changes for a dataset. The daql command
should be able to generate simple rules and migration templates.
