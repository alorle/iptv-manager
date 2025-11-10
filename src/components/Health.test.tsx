import { describe, expect, it, vi } from "vitest";

import HealthCheck from "./Health";

import { $api } from "@/lib/api/client";
import { renderWithProviders, screen } from "@/test/utils";

// Mock the API client
vi.mock("@/lib/api/client", () => ({
  $api: {
    useQuery: vi.fn(),
  },
}));

const mockUseQuery = vi.mocked($api.useQuery);

describe("HealthCheck", () => {
  it("displays loading state initially", () => {
    mockUseQuery.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText(/checking system health/i)).toBeInTheDocument();
    expect(screen.getByText(/iptv manager/i)).toBeInTheDocument();
  });

  it("displays error state when API call fails", () => {
    mockUseQuery.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText(/unhealthy/i)).toBeInTheDocument();
    expect(screen.getByText(/failed to connect to api/i)).toBeInTheDocument();
  });

  it("displays health data when API call succeeds", () => {
    const mockData = {
      status: "healthy",
      version: "1.0.0",
      timestamp: "2025-01-10T12:00:00Z",
    };

    mockUseQuery.mockReturnValue({
      data: mockData,
      isLoading: false,
      isError: false,
      dataUpdatedAt: new Date("2025-01-10T12:00:00Z").getTime(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText(/healthy/i)).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText(/live/i)).toBeInTheDocument();
  });

  it("displays version information", () => {
    const mockData = {
      status: "healthy",
      version: "2.3.4",
      timestamp: "2025-01-10T12:00:00Z",
    };

    mockUseQuery.mockReturnValue({
      data: mockData,
      isLoading: false,
      isError: false,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("2.3.4")).toBeInTheDocument();
  });

  it("displays server time", () => {
    const timestamp = "2025-01-10T12:00:00Z";
    const mockData = {
      status: "healthy",
      version: "1.0.0",
      timestamp,
    };

    mockUseQuery.mockReturnValue({
      data: mockData,
      isLoading: false,
      isError: false,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText("Server Time")).toBeInTheDocument();
    // The actual formatted date will depend on locale
    const serverTimeElement = screen.getByText("Server Time").nextElementSibling;
    expect(serverTimeElement).toBeInTheDocument();
  });

  it("displays last check time", () => {
    const mockData = {
      status: "healthy",
      version: "1.0.0",
      timestamp: "2025-01-10T12:00:00Z",
    };

    const dataUpdatedAt = new Date("2025-01-10T12:00:00Z").getTime();

    mockUseQuery.mockReturnValue({
      data: mockData,
      isLoading: false,
      isError: false,
      dataUpdatedAt,
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText("Last Check")).toBeInTheDocument();
  });

  it("configures refetch interval", () => {
    mockUseQuery.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    // Verify that useQuery was called with correct parameters
    expect(mockUseQuery).toHaveBeenCalledWith("get", "/health", {}, { refetchInterval: 5000 });
  });

  it("converts status to title case", () => {
    const mockData = {
      status: "healthy",
      version: "1.0.0",
      timestamp: "2025-01-10T12:00:00Z",
    };

    mockUseQuery.mockReturnValue({
      data: mockData,
      isLoading: false,
      isError: false,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof $api.useQuery>);

    renderWithProviders(<HealthCheck />);

    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });
});
