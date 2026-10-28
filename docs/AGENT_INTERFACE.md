# Agent interface

Normal commands support `--output json` and `--non-interactive`. JSON implies
noninteractive execution. A nonterminal stdin also disables prompts. Terminal
users keep table output and selection prompts; the same typed data drives both
renderers.

```sh
cli-top schema
cli-top schema marks
cli-top semesters --output json
cli-top marks --semester-id PORTAL_SEMESTER_ID --output json
```

`schema`, help, version, and completion work without credentials. Schema/help/
completion construction performs no configuration writes or startup network
requests. Invalid command syntax and conflicting selectors are checked before
policy checks or authentication. Portal commands still honor client policy.

## Result protocol

Each normal JSON command writes one JSON document to stdout, including failures:

```json
{
  "schema_version": "1",
  "command": "marks",
  "success": true,
  "data": {
    "courses": [{
      "code": "CSE1001",
      "title": "Data Structures",
      "type": "Theory Only",
      "faculty": "Example Faculty",
      "slot": "A1",
      "assessments": [{
        "title": "CAT 1",
        "maximum": 50,
        "weight": 15,
        "status": "Present",
        "score": 45,
        "weighted_score": 13.5
      }]
    }]
  },
  "context": {
    "semester": {"id": "PORTAL_SEMESTER_ID", "label": "Fall", "scope": "academic", "index": 1},
    "selection_source": "explicit_id",
    "timezone": "Asia/Kolkata"
  },
  "warnings": [],
  "error": null
}
```

Numbers are JSON numbers. Missing numeric values and unavailable portal IDs are
`null`. List results use arrays. JSON contains domain fields rather than table
snapshots. Debug output, progress messages, credential diagnostics, and terminal
colors do not enter the JSON stream. Raw exceptions are deliberate: `schema`
always emits its schema/catalog JSON directly; help and completion emit text or
shell scripts.

Failures expose a stable `error.kind`, sanitized `message`, `retryable`, and,
where applicable, `http_status`, `required_flags`, `choices`, and `next` actions.
Internal error causes are retained for Go callers but are not serialized.

| Exit | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Internal or local I/O failure |
| 2 | Invalid arguments or selection |
| 3 | Authentication required or failed |
| 4 | Selection, confirmation, or human action required |
| 5 | Network, upstream, or parse failure |
| 6 | Policy block or decommissioned client |
| 7 | Partial failure; inspect data and warnings |
| 130 | Cancellation |

A failed candidate semester or calendar month cannot silently become a complete
success. Planners retain available classes/exams, set `complete: false`, and
report warnings when a dependency fails. Attendance is joined by course code and
lab/theory type. An ambiguous match leaves `can_skip` null rather than guessing.

## Discovering scoped IDs

```sh
cli-top semesters --output json
cli-top class-groups --semester-id SEM_ID --output json
cli-top calendar --semester-id SEM_ID --class-group-id GROUP_ID --output json

cli-top courses --source consolidated --output json
cli-top faculties --course-id COURSE_ID --output json
cli-top course-page --course-id COURSE_ID --faculty-id ERP_ID --output json

cli-top courses --source archive --semester-id SEM_ID --output json
cli-top faculties --source archive --semester-id SEM_ID --course-id CLASS_ID --output json

cli-top courses --source syllabus --category-id CATEGORY_ID --output json
cli-top courses --source allocation --category-id CATEGORY_ID --output json
cli-top courses --source assignments --semester-id SEM_ID --output json
```

IDs are opaque and belong to a source and parent scope. Academic semesters and
the semester labels embedded in consolidated course pages are different
namespaces. The consolidated page supports its existing local `--semester`
index; it does not accept an academic `--semester-id`. Archive faculty choices
include both ERP and class identity. Use the exact ID returned by discovery.
Attendance and timetable pages do not reliably expose a portal course ID, so
those fields remain null; the course code is a separate field.

Legacy index/query flags remain supported. Indexes start at 1 and can change
when VTOP reorders options. Exact ID selectors cannot be combined with their
legacy selector. An unknown ID never falls back to a fuzzy name match. Queries
resolve a unique case-insensitive label or substring; ambiguous queries return
choices. A single available option can be resolved automatically.

For example, `marks --output json` with multiple semesters returns exit 4,
`required_flags: ["--semester-id"]`, semester `choices`, and executable `next`
actions for schema and discovery. Rerun with one of the returned IDs. Dynamic
curriculum categories and application choices are also returned in selection
errors; discovering them does not submit anything.

`schema COMMAND` describes actual Cobra flags, aliases, defaults, selector
conflicts, conditional requirements, authentication, effects, and a JSON Schema
for the result envelope and concrete domain DTO. Schema inspection is offline;
it does not fetch a user's dynamic choices.

## Actions and artifacts

```sh
cli-top course-page --course-id COURSE_ID --faculty-id ERP_ID --materials 1,2-4 --output json
cli-top syllabus --category-id CATEGORY_ID --course-id COURSE_CODE --output json
cli-top da --semester-id SEM_ID --course-id CLASS_ID --assignment-id ASSIGNMENT_ID --output json
cli-top events --event-id EVENT_ID --confirm --output json
cli-top facility --facility-id FACILITY_ID --confirm --output json
```

Course-page commands list material metadata until `--materials` is supplied;
`0` selects all downloadable materials. Downloads return `files` with absolute
paths and save status. Syllabus and explicit assignment selection download the
selected document. Assignment IDs are scoped to a course and can be null when
VTOP does not expose them; the legacy assignment query/index remains available.
Downloads do not open folders, launch viewers, or upload calendars.

`events`, `facility`, `leave`, and `nightslip` without action flags read status.
Event/facility registration requires confirmation. Leave/nightslip submission
requires `--apply` and application fields; `schema` lists the required inputs.
A submission whose outcome cannot be verified returns an error instructing the
caller to inspect status before retrying. It is not marked retryable.

`login` stores encrypted local credentials; it does not claim a VTOP session has
been authenticated. Portal commands authenticate lazily. Manual authentication
requirements return `human_action_required` in machine mode. Configuration
writes use a temporary file with mode 0600 and atomic replacement.

## Compatibility and implementation notes

Command names, aliases, and legacy selector flags remain available. Presentation
now comes from typed results, so table layouts can differ. Read commands no
longer incidentally generate/upload ICS files. Status commands require explicit
action flags for submissions instead of offering a follow-on application flow.
The semester cache remains in memory during typed command invocations.

The hidden `proxy USER PASSWORD COMMAND ...` interface remains an adapter over
the same operations. Existing response fields (`output`, `structured_data`,
timestamps, etc.) remain; structured results add `data`, `context`, `warnings`,
and `schema_version`. Presentation tables/messages are generated from domain
objects without swapping process stdout. Table layouts can differ. Errors now
produce nonzero exits; callers that previously assumed exit 0 must handle this.
`sync` retains per-command results and returns exit 7 if any command fails. It
accepts read operations only.

The command tree and flags are fresh per invocation. A process-wide mutex
serializes the embedded `Run` API while legacy helper session/presentation state
still exists. This implementation does not claim concurrent embedded sessions.
Legacy exported feature functions remain for existing Go callers; CLI commands
use `features.Session.Execute` and the typed contract.
