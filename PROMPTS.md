# How this project was built

This calculator was built in a conversation with an AI coding assistant.
This file records what was asked, in order: first a short summary, then the
prompts verbatim.

## Summary

1. **Initial build.** A full-stack calculator: a React + TypeScript frontend,
   a Go REST backend, unit tests with coverage, a README, and one Dockerfile
   for both.
2. **REST and engineering practices.** Resource-oriented endpoints, RFC 9457
   style errors, an OpenAPI contract, request IDs, panic recovery, security
   headers, linting and CI.
3. **Correct results, no races.** The latest request always wins; earlier or
   late responses never replace newer results. Race tests on both layers.
4. **Windows-style UI.** A keypad calculator modelled on the Windows Standard
   calculator, without memory keys.
5. **Cleanup.** Redundant code removed, lint passing everywhere, a minimal API
   contract: `{"result": n}` on success, `{"code", "detail"}` on error.
6. **API docs.** The OpenAPI spec served by the backend, and interactive docs
   at `/docs/`.
7. **Clearer error documentation.** A specific description for every error
   response, briefly referencing RFC 9457.
8. **Percent.** `%` became a two-number "a percent of b" operator, and negative
   percentages are rejected.
9. **Errors and tests.** The UI shows the backend's error messages (for example
   for a negative square root), and the calculator tests cover the full
   float64 range, plus fuzzing.
10. **History.** Every calculated result is recorded, including repeated √ and
    x²; history stays in memory only.
11. **Timeout.** Requests time out after 1 second.

## Prompts

Prompts are reproduced as typed. Replies to on-screen choices are included
as the text they sent.

### 1

> Objective
> Build a full-stack calculator application with a React frontend and a backend microservice. The frontend should consume the backend API to perform basic and advanced arithmetic operations. Focus on clean design, maintainable code, and testable architecture.Requirements
> Functional
> Operations:
> - Addition, Subtraction, Multiplication, Division
> - Optional: Exponentiation, Square Root, Percentage
> Frontend (React):
> - Intuitive UI for entering input and displaying results
> - Input validation and error handling
> - Responsive design (basic mobile support)
> Backend (REST API):
> - Expose endpoints for calculator operations
> - Validate input and handle edge cases (division by zero, invalid data)
> - Return results in JSON format
> Non-Functional
> - Clean, readable, and idiomatic code (frontend and backend)
> - Unit tests covering key functionality for both layers
> - Documentation: setup instructions, API usage, and design rationale
> - Dockerfile for full-stack deployment
> Constraints
> - Frontend: React TypeScript
> - Backend: Go
> Deliverables
> 1. Git repository with frontend and backend code
> 2. README with setup instructions, API examples, and design decisions
> 3. Unit tests and coverage report
> 4. Dockerfile to run frontend + backend together
> Instructions
>
> 2.  Prioritize correctness, clarity, and maintainability over extra features.
> 6. Make sure your README includes:
> - Setup instructions
> - How to run the frontend and backend
> - Examples of API calls (if using REST)
> - Design decisions or assumptions
>
> Create a separate subfolder calculator for this.

### 2

> make sure to use rest API, and act according to best engineering practices

### 3

> please run the service so that I can test it; also make sure previous results do not supersede the current ones

### 4

> also make sure there are no race conditions

### 5

> and please make a more relatable UI: like a Windows calculator app, but without memory

### 6

> and then remove all the redundant code, make sure everything follows frontend and backend best practices and lint is passing, and the api has clear boundaries, rest architecture and no redundant return values. Keep the app running so that I can test it.

### 7

> is the api definition available via the server endpoint? I want to see the docs

### 8

> The docs mention "405
> Error described as RFC 9457 Problem Details." That's not very helpful. Can you quote the relevant part of the RFC? Especially since there are several such documentation mentions.

### 9

> Okay. Let's forbid negative number of percents; and the UI does not allow me to enter 2 numbers and get a % of b. Pressing the % just divides my input by 100. Fix that.

### 10

> let's delete the overly verbose references to RFC from Problem API response. just mention it's based on rfc, that's fine; don't explain how every detail corresponds to the phrasing from the rfc
>
> trying to square root of -5 gives invalid input, although the API (presumably) returns NEGATIVE_SQUARE_ROOT. Can you check whether it's the case, and propagate such informative errors to the UI?
>
> add more tests to calculator_test.go, to make the calculations bulletproof, and working for both small and big numbers
>
> and explain to me the Frontend part, I ve never worked with react before

### 11

> clicking on square root repeatedly gives me the result, but doesn't put it into hustory. Please fix that, and check if it happens for different operations asw ell. And how does history work, is it backend of frontend?

### 12

> No, keep history in memory only.

### 13

> can we wait 1s max for the response? 10s sems too big

