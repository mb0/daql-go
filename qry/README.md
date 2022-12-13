qry
===

Package qry provides a way to describe and evaluate queries for local and external data.

We use a regular xelf program and mark it as query aware by providing a program specific `Doc`
environment. The doc envirinment provides a `Backend`, resolves special query subjects to a query
`Spec` with model details and stores references to all query jobs.

The query `Spec`, when resolved, creates a `Job` environment, registers it with the doc environment
and then proceeds to resolves its arguments to build a `Task` with all relevant query details. When
evaluated the spec calls the backend and returns the result.

The doc and job environments together provide access to all query tasks and results.

We also automatically provide a dom backend to query the project, schemas and models of the project.
