import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { Boss2Page } from "./Boss2Page";

describe("Boss2Page", () => {
	it("renders the boss title", () => {
		render(
			<MemoryRouter>
				<Boss2Page />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("Boss 2: State Mismatch (CSRF)"),
		).toBeInTheDocument();
	});

	it("renders authorization buttons", () => {
		render(
			<MemoryRouter>
				<Boss2Page />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("Authorize WITHOUT state (Vulnerable)"),
		).toBeInTheDocument();
		expect(
			screen.getByText("Authorize WITH state (Protected)"),
		).toBeInTheDocument();
	});

	it("mentions RFC 6749", () => {
		render(
			<MemoryRouter>
				<Boss2Page />
			</MemoryRouter>,
		);
		expect(screen.getByText(/RFC 6749/)).toBeInTheDocument();
	});
});
