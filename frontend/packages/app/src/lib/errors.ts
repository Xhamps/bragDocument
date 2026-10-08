import { ApiError } from "./api";

/** Human-readable text for a thrown mutation/query error; 422 fields as "field: message". */
export function errorText(e: unknown) {
  if (e instanceof ApiError)
    return e.fields
      ? Object.entries(e.fields)
          .map(([field, message]) => `${field}: ${message}`)
          .join(", ")
      : e.message;
  return e instanceof Error ? e.message : null;
}
