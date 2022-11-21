After working on an javascript app and form library for daql some sore patches were noticed.

Query Result Types
------------------

Done. We now send the query result type with the query data, so we can reliably and correctly
parse time, enum and bit values. This is very short for schema types. We can then use the registry
to construct user defined types for named types like enums.

Refence Types
-------------

Done. All primary keys and references keep the reference name now. 
With that in place we can easily handle partial model query results. We can now simply return
filtered object type and have the id identify the models.

App Config
----------

We have a project dom node that can hold configuration common to a whole project. But often enough
a project has multiple applications that each cover a different part of the project and each have
their own config parameters.

We want app dom nodes that specialize and extend the project config. We would have one parent
project that includes multiple applications nodes. For example if we want to use a central server
and different satellite applications for a POS-terminals and the office PCs, we would add three
applications that can exclude certain project schemas, models or fields.

Application resolution should check if the restricted schema is still valid. If we remove a model
we must also check all possible reference to that model in remaining models.

Daql Modules
------------

Xelf now supports a module system to extend a programs environment. We want to provide custom module
sources for the dom and qry pacakges, to enable their setup dynamically.

A dom module source is already implemented and provides its own schema and model types. The qry
module should ensure or load the dom module and setup the doc environment.

To configure the qry doc we need to setup a backend implementation, that in turn requires a dom
project. The idea would be to export and register projects in the dom module, and register backend
providers by uri scheme with the qry mod and provide a spec in the qry module to setup the backend.

The bend spec to setup the backend takes a uri to identify the backend and data source and
optionally the project dom or later an app dom node to provide, if the project is omitted the
last registered project is used.

	(use 'daql/qry' 'myproject')
	(qry.bend 'postgres:///daql')

The uri scheme represents the backend implementation and the rest of the uri the data source.
Combined backends can themselves registered with a new url scheme. And the url path query is
flexible enough to encode the necessary information.

 (I mulled over failing when loading more than one project into a program, but feel bad about it.
 Loading mutliple projects could mostly be usefull for templates or importing groups of schemas into
 the main project, and therefor it is easier to arange for the main project to be last.)

Endpoint generation
-------------------

It would be nice to generate query endpoints, server and client type definitions from daql schemas.

We could add named query nodes to the project or schemas and the code gen can resolve the query
result types and generate the code we want.

We could see the query as view model, similar to how databases use views. We could also use these
views to restrict field access per project or dynamically by role. We could then use the same query
syntax to work with these view models and restrict or extend fields on a per query basis, just like
we allow with models now.

Model Labels
------------

As a convention schemas, models and elements should use a label tag that either holds the
translation directly for single language deployments, or a key used by an external localization
library. Similar a descr tag with more details. The user should decide what strategy and l10n keys
to use and therefor must be able to annotate loaded schemas with these tags.

Schema Modification
-------------------

We want to define schemas centrally as part of libraries, but allow the user to filter or extend
these schemas per project. We need that most importantly to direct code generation in a declarative
way using custom code generation node flags.

We may want to be able to add a private field to a model type to cache a computed property or
hold an internal resource handle, we may want to change labels or add custom flags for special
user needs.

Generators Plug-ins
-------------------

Code generation should be easily extensible to avoid excessive configuration flags and encourage
writing custom generators instead. We have however a daql gen command and it would be nice if we
can make use of simple custom code generators there. Because code gen is not run often we could
allow exec external commands as option, similar to what go generate does. Then we can just add a
couple of go files and call go run. We can even call go generate as part of the code gen.

We could add a option to pass the json serialized project to generator commands (as to not resolve
the daql project for each generator). But it should be an option so we exec any command that
does not expect project data on stdin (like go generate) to kick of related workflows.
