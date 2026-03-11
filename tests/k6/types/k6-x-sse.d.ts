// tests/k6/types/k6-x-sse.d.ts
declare module "k6/x/sse" {
  type SSEEvent = { name?: string; data?: string };
  type SSEClient = {
    on(event: "open" | "error" | "event", handler: (event: SSEEvent) => void): void;
    close(): void;
  };
  type SSEResponse = { status: number };
  const sse: {
    open(
      url: string,
      params: Record<string, any>,
      cb: (client: SSEClient) => void
    ): SSEResponse;
  };
  export default sse;
}
