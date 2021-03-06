qry
===

Package qry provides a way to describe and evaluate queries for local and external data.

We use a global `Qry` context to execute queries. The query context has everything that is needed to
evaluate queries. That includes the program environment and a query `Backend`.

To execute a query program we provide a `Doc` root environment that resolves special symbols to a
query `Subj` with model and backend details and then constructs and returns a query spec.

The query `Spec`, when resolved, creates a `Job` environment, registers it with the doc environment
and then proceeds to resolves its arguments to build a `Task` with all relevant query details. When
evaluated the spec calls the subject backend and returns the result.

The doc and job environments together provide access to all query tasks and results.

We also automatically provide a dom backend to query the project, schemas and models of the project.
