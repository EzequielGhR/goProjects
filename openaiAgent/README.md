# DESCRIPTION
A simple agent project made to interactively follow arize-ai's Evaluating AI Agents course, while also learning Go.
Implements chat completion using OpenAI, Tracing with OpenTelemetry, plus some basic replications to arize-ai's openinference spans
to properly display traces and spans in phoenix.

# BUILD
- You need to have a few phoenix credentials on your environment variables: 'PHOENIX_COLLECTOR_ENDPOINT' and 'PHOENIX_CLIENT_HEADERS', both can be found on your phoenix free account.
- Run build.sh and it should compile to bin/v1/main.o

# RUN
Run run.sh with your prompt as positional argument. If it isn't compiled already, it will do it before running.
The project path is set as "../..", so its important to have it set up in a structure similar, or change the source code to your liking.

# Structure
The whole project structure is divided into 4 modules
- The main module: Handles user input and starts main span before running the agent.
- The agent module: Handles everything agent related, plus the main logic to handle tool calls.
- The tools module: All the tools logic can be found here.
- The trace module: Helper functions and types for easily handling openinference-like spans, tracer providers, and other telemetry stuff.

The whole chain of calls and spans function as follows:
```
    Agent
    ├── RouterCalls
    |       ├── ChatCompletion
    │       ├── HandleToolCalls
    │       │   ├── ToolCall
    |       |   |   ├── StepCall
    |       |   |   |   ├── Chat Completion
    |       |   |   |   └── Chat Completion
    |       |   |   └── StepCall
    |       |   |       └── ChatCompletion
    │       │   ├── ToolCall
    |       |   |   └── StepCall
    │       │   └── ToolCall
    │       └── HandleToolCalls
    │           ├── ToolCall
    |           |   └── ChatCompletion
    │           ├── ToolCall
    │           └── ToolCall
    |               └── StepCall
    └── RouterCalls
        └── ...
```
The context of each span is tracked via global variables and carried over to each child if any.
You should be able to see the traces on Phoenix, here is an example:

![](trace_details.png)


# Important notes:

- The spans started from the tracer provider mimic openinference spans, so special functions have been designed to handle them, as well as types and constants
```
    func StartOpenInferenceSpan(
        spanName string,
        openInferenceSpanKind OpenInferenceSpanKind,
        parentSpanContext context.Context,
    ) (context.Context, trace.Span) {
        if parentSpanContext == nil {
            parentSpanContext = context.Background()
        }

        ctx, span := GetActiveTracer().Start(
            parentSpanContext,
            spanName,
            trace.WithSpanKind(trace.SpanKindInternal),
            trace.WithAttributes(
                attribute.String(openInferenceSpanKindKey, strings.ToUpper(string(openInferenceSpanKind))),
            ),
        )

        log.Printf("Starting '%s' OpenInference Span with kind '%s'\n", spanName, openInferenceSpanKind)
        return ctx, span
    }
```
