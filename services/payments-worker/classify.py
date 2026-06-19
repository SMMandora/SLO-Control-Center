"""Pure SLO-classification helper for processed payments."""


def classify(enqueued_ms: int, done_ms: int, ok: bool) -> tuple[str, float]:
    """Return (status, end_to_end_latency_seconds) for a processed payment.

    status is "success" when the gateway charge succeeded, else "failure".
    latency is clamped to >= 0 to tolerate minor clock skew.
    """
    latency_s = max(0.0, (done_ms - enqueued_ms) / 1000.0)
    status = "success" if ok else "failure"
    return status, latency_s
