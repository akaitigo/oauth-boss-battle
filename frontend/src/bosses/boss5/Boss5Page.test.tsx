import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { Boss5Page } from "./Boss5Page";

describe("Boss5Page", () => {
	it("renders the boss title", () => {
		render(
			<MemoryRouter>
				<Boss5Page />
			</MemoryRouter>,
		);
		expect(screen.getByText("Boss 5: Logout Hell")).toBeInTheDocument();
	});

	it("renders login button", () => {
		render(
			<MemoryRouter>
				<Boss5Page />
			</MemoryRouter>,
		);
		expect(screen.getByText("Login (SSO)")).toBeInTheDocument();
	});

	it("mentions Back-Channel Logout", () => {
		render(
			<MemoryRouter>
				<Boss5Page />
			</MemoryRouter>,
		);
		expect(screen.getByText(/Back-Channel Logout/)).toBeInTheDocument();
	});
});
