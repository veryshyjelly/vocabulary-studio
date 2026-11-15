import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SessionPersistenceNotice } from "./SessionPersistenceNotice";
import { SessionStoreMode, type AccountStatusDTO } from "../../lib/bridge";

const auth = vi.hoisted(() => ({ GetPersistenceStatus: vi.fn(), RetryPersistence: vi.fn() }));
vi.mock("../../lib/bridge", async (original) => {
  const actual = await original<typeof import("../../lib/bridge")>();
  return { ...actual, getBridge: () => ({ ...actual.getBridge(), auth }) };
});

const account = (state = "failed", uid = "learner"): AccountStatusDTO => ({
  authenticated: true, valid_user: true, uid, nickname: "Learner", points: 0,
  level: { id: "L01", name: "Novice", milestone: 0, progress: 0 }, paid: false,
  full_vocab_trainer: false, session_store_mode: SessionStoreMode.SessionStoreFile,
  persistence_state: state,
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => { resolve = res; });
  return { promise, resolve };
}
const warning = /signed in, but your session could not be saved/;

describe("session persistence feedback", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    auth.GetPersistenceStatus.mockResolvedValue({ state: "failed" });
    auth.RetryPersistence.mockResolvedValue({ state: "saved" });
  });
  afterEach(() => vi.useRealTimers());

  it("keeps authenticated partial success visible and retries the selected backend", async () => {
    render(<SessionPersistenceNotice account={account()} />);
    await act(async () => {});
    expect(screen.getByText(warning)).toBeInTheDocument();
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Retry saving session" })); });
    expect(auth.RetryPersistence).toHaveBeenCalledExactlyOnceWith();
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  it("shows only safe generic feedback when retry fails", async () => {
    auth.RetryPersistence.mockRejectedValue(new Error("raw cookie credential detail"));
    render(<SessionPersistenceNotice account={account()} />);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Retry saving session" })); });
    expect(screen.getByText(warning)).toBeInTheDocument();
    expect(screen.getByText(/still could not be saved/)).toBeInTheDocument();
    expect(screen.queryByText(/raw cookie/)).not.toBeInTheDocument();
  });

  it("detects later refresh persistence failures by polling local status", async () => {
    auth.GetPersistenceStatus.mockResolvedValueOnce({ state: "saved" }).mockResolvedValue({ state: "failed" });
    render(<SessionPersistenceNotice account={account("saved")} />);
    await act(async () => {});
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(screen.getByText(warning)).toBeInTheDocument();
    expect(auth.GetPersistenceStatus).toHaveBeenCalledTimes(2);
  });

  it("clears old retry feedback after a successful poll before a later failure", async () => {
    auth.RetryPersistence.mockRejectedValue(new Error("save failed"));
    auth.GetPersistenceStatus.mockResolvedValueOnce({ state: "failed" })
      .mockResolvedValueOnce({ state: "saved" }).mockResolvedValue({ state: "failed" });
    render(<SessionPersistenceNotice account={account()} />);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Retry saving session" })); });
    expect(screen.getByText(/still could not be saved/)).toBeInTheDocument();
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(screen.getByText(warning)).toBeInTheDocument();
    expect(screen.queryByText(/still could not be saved/)).not.toBeInTheDocument();
  });

  it("does not clear known failure when local status is unknown", async () => {
    auth.GetPersistenceStatus.mockResolvedValue({ state: "unknown" });
    auth.RetryPersistence.mockResolvedValue({ state: "unknown" });
    render(<SessionPersistenceNotice account={account()} />);
    await act(async () => {});
    expect(screen.getByText(warning)).toBeInTheDocument();
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Retry saving session" })); });
    expect(screen.getByText(warning)).toBeInTheDocument();
  });

  it("does not let an older status poll undo a successful retry", async () => {
    const poll = deferred<{ state: string }>();
    auth.GetPersistenceStatus.mockReturnValueOnce(poll.promise);
    render(<SessionPersistenceNotice account={account()} />);
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Retry saving session" })); });
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
    await act(async () => poll.resolve({ state: "failed" }));
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  it("clears feedback for deliberate memory mode and ignores pending polls", async () => {
    const poll = deferred<{ state: string }>();
    auth.GetPersistenceStatus.mockReturnValueOnce(poll.promise);
    const view = render(<SessionPersistenceNotice account={account()} />);
    view.rerender(<SessionPersistenceNotice account={{ ...account(), session_store_mode: SessionStoreMode.SessionStoreMemory }} />);
    await act(async () => { poll.resolve({ state: "failed" }); await vi.advanceTimersByTimeAsync(10000); });
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
    expect(auth.GetPersistenceStatus).toHaveBeenCalledTimes(1);
  });

  it("ignores previous-account responses and stops polling after sign-out", async () => {
    const poll = deferred<{ state: string }>();
    auth.GetPersistenceStatus.mockReturnValueOnce(poll.promise).mockResolvedValue({ state: "saved" });
    const view = render(<SessionPersistenceNotice account={account()} />);
    view.rerender(<SessionPersistenceNotice account={account("saved", "other-learner")} />);
    await act(async () => poll.resolve({ state: "failed" }));
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
    view.rerender(<SessionPersistenceNotice account={null} />);
    const calls = auth.GetPersistenceStatus.mock.calls.length;
    await act(async () => { await vi.advanceTimersByTimeAsync(10000); });
    expect(auth.GetPersistenceStatus).toHaveBeenCalledTimes(calls);
  });

  it("ignores late retries for a replaced account", async () => {
    const retry = deferred<{ state: string }>();
    auth.RetryPersistence.mockReturnValueOnce(retry.promise);
    auth.GetPersistenceStatus.mockResolvedValueOnce({ state: "failed" }).mockResolvedValue({ state: "saved" });
    const view = render(<SessionPersistenceNotice account={account()} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry saving session" }));
    view.rerender(<SessionPersistenceNotice account={account("saved", "other-learner")} />);
    await act(async () => retry.resolve({ state: "failed" }));
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  it("cleans up polling on unmount", async () => {
    const view = render(<SessionPersistenceNotice account={account()} />);
    await act(async () => {});
    view.unmount();
    await act(async () => { await vi.advanceTimersByTimeAsync(10000); });
    expect(auth.GetPersistenceStatus).toHaveBeenCalledTimes(1);
  });
});
