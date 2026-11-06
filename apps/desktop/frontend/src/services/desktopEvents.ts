import { Events } from "@wailsio/runtime";
import type {
  ChangedEventDTO,
  DecisionRequiredDTO,
  ProgressEventDTO,
} from "../features/downloads/downloadTypes";

export const EVENT_PROGRESS = "odm:download-progress";
export const EVENT_CHANGED = "odm:download-changed";
export const EVENT_DECISION = "odm:decision-required";

export type Unsubscribe = () => void;

export function subscribeProgress(
  handler: (ev: ProgressEventDTO) => void,
): Unsubscribe {
  return Events.On(EVENT_PROGRESS, (e) => {
    handler(e.data as ProgressEventDTO);
  });
}

export function subscribeChanged(
  handler: (ev: ChangedEventDTO) => void,
): Unsubscribe {
  return Events.On(EVENT_CHANGED, (e) => {
    handler(e.data as ChangedEventDTO);
  });
}

export function subscribeDecision(
  handler: (ev: DecisionRequiredDTO) => void,
): Unsubscribe {
  return Events.On(EVENT_DECISION, (e) => {
    handler(e.data as DecisionRequiredDTO);
  });
}
