from classify import classify


def test_success_latency():
    status, latency = classify(1000, 3000, True)
    assert status == "success"
    assert latency == 2.0


def test_failure_status():
    status, _ = classify(1000, 2000, False)
    assert status == "failure"


def test_clamps_negative_skew():
    _, latency = classify(5000, 1000, True)
    assert latency == 0.0
