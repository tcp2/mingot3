import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Api, ConfigBuilder } from "../../src/backend/services/orbita";

// Mock globals
const _originalFetch = global.fetch;

describe("ConfigBuilder", () => {
	describe("buildIntl", () => {
		it("should extract language and timezone correctly", () => {
			const profile = { navigator: { language: "en-US,en;q=0.9" } };
			const tz = {
				timezone: "America/New_York",
				languages: "en-US,en",
				country: "US",
			};
			const intl = ConfigBuilder.buildIntl(profile, tz);
			expect(intl.accept_languages).toBe("en-US-US,en,en-US");
			expect(intl.app_locale).toBe("en");
		});

		it("should fallback to lang if autoLang is false", () => {
			const profile = { navigator: { language: "vi-VN" }, autoLang: false };
			const tz = { timezone: "", isTimezone: false };
			const intl = ConfigBuilder.buildIntl(profile, tz);
			expect(intl.accept_languages).toBe("vi-VN");
		});
	});

	describe("buildPrefs", () => {
		it("should generate correct preferences with valid proxy", () => {
			const profile = {
				id: "profile123",
				navigator: {
					userAgent: "TestUA",
					resolution: "1920x1080",
					doNotTrack: true,
				},
				webGLMetadata: {
					mode: "noise",
					vendor: "Google",
					renderer: "SwiftShader",
				},
				proxy: { username: "user", password: "pwd" },
				name: "Test Profile",
			};

			const prefs = ConfigBuilder.buildPrefs(
				profile,
				{},
				"user123",
				"localSession",
				"Test Profile Name",
			);

			expect(prefs.profile_id).toBe("profile123");
			expect(prefs.userAgent).toBe("TestUA");
			expect(prefs.proxy.username).toBe("user");
			expect(prefs.proxy.password).toBe("pwd");
		});

		it("should handle disabled proxy correctly", () => {
			const profile = {
				navigator: { userAgent: "TestUA", resolution: "1920x1080" },
			};
			const prefs = ConfigBuilder.buildPrefs(
				profile,
				{},
				"user123",
				"localSession",
				"Test Profile Name",
			);
			expect(prefs.proxy.username).toBe("");
		});
	});
});

describe("Api", () => {
	beforeEach(() => {
		vi.stubGlobal("fetch", vi.fn());
		vi.spyOn(console, "log").mockImplementation(() => {});
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	describe("request", () => {
		it("should return JSON on success", async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				status: 200,
				text: async () => JSON.stringify({ success: true }),
				json: async () => ({ success: true }),
			});
			const data = await Api.request(
				"GET",
				"/test",
				{ Authorization: "Bearer token" },
				{ body: "data" },
			);
			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining("/test"),
				expect.objectContaining({
					method: "GET",
					headers: expect.objectContaining({
						Authorization: "Bearer token",
					}),
				}),
			);
			expect(data).toEqual({ success: true });
		});

		it("should throw error on 403", async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 403,
				text: async () => "Forbidden",
				json: async () => ({ error: "Forbidden" }),
			});
			await expect(
				Api.request("GET", "/test", { Authorization: "Bearer token" }),
			).rejects.toThrow("HTTP 403: Forbidden");
		});
	});

	describe("fetchProfileToken", () => {
		it("should extract profile_token correctly", async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				status: 200,
				text: async () => JSON.stringify({ token: "jwt-token-123" }),
				json: async () => ({ token: "jwt-token-123" }),
			});
			const token = await Api.fetchProfileToken("profile123", "auth-token");
			expect(token).toBe("jwt-token-123");
		});
	});
});
