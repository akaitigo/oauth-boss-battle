import { useState } from "react";
import { useBossProgress } from "../../hooks/useBossProgress";
import type { BossResult } from "../../types/boss";

const API_BASE = "/api/boss/3";

type Phase = "intro" | "authorize" | "token" | "replay" | "result";

interface TokenResponse {
	success: boolean;
	defeated?: boolean;
	id_token?: string;
	message?: string;
	explanation?: string;
	rfc_link?: string;
}

export function Boss3Page() {
	const { isDefeated, markDefeated } = useBossProgress();
	const [phase, setPhase] = useState<Phase>("intro");
	const [authCode, setAuthCode] = useState("");
	const [nonceValue, setNonceValue] = useState("");
	const [idToken, setIdToken] = useState("");
	const [result, setResult] = useState<BossResult | null>(null);
	const [tokenResponse, setTokenResponse] = useState<TokenResponse | null>(
		null,
	);

	const defeated = isDefeated(3);

	const handleAuthorizeWithoutNonce = async () => {
		const res = await fetch(`${API_BASE}/authorize`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				client_id: "demo-client",
				redirect_uri: "http://localhost:3000/callback",
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.code) setAuthCode(data.code);
		setNonceValue("");
		setPhase("authorize");
	};

	const handleAuthorizeWithNonce = async () => {
		const nonceRes = await fetch(`${API_BASE}/generate-nonce`, {
			method: "POST",
		});
		const nonceData: { nonce: string } = await nonceRes.json();
		setNonceValue(nonceData.nonce);

		const res = await fetch(`${API_BASE}/authorize`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				client_id: "demo-client",
				redirect_uri: "http://localhost:3000/callback",
				nonce: nonceData.nonce,
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.code) setAuthCode(data.code);
		setPhase("authorize");
	};

	const handleTokenExchange = async (withNonceValidation: boolean) => {
		const body: Record<string, string> = { code: authCode };
		if (withNonceValidation && nonceValue) {
			body.expected_nonce = nonceValue;
		}

		const res = await fetch(`${API_BASE}/token`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(body),
		});
		const data: TokenResponse = await res.json();
		setTokenResponse(data);
		if (data.id_token) setIdToken(data.id_token);
		if (data.defeated) markDefeated(3);
		setPhase("token");
	};

	const handleReplay = async () => {
		const res = await fetch(`${API_BASE}/replay`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				id_token: idToken,
				expected_nonce: nonceValue ? "different-session-nonce" : "",
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		setPhase("replay");
	};

	const handleReset = () => {
		setPhase("intro");
		setAuthCode("");
		setNonceValue("");
		setIdToken("");
		setResult(null);
		setTokenResponse(null);
	};

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>Boss 3: Nonce Replay Attack</h1>
			<p>
				ID Token replay via nonce reuse/absence — OpenID Connect Core Section
				11.5 requires a unique nonce per authentication session.
			</p>

			{defeated && (
				<div
					style={{
						background: "#d4edda",
						border: "1px solid #c3e6cb",
						padding: 16,
						borderRadius: 8,
						marginBottom: 16,
					}}
				>
					Boss Defeated!
				</div>
			)}

			{phase === "intro" && (
				<div>
					<h2>Step 1: Authorization Request</h2>
					<p>Choose how to start the OIDC authentication flow:</p>
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={handleAuthorizeWithoutNonce}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITHOUT nonce (Vulnerable)
						</button>
						<button
							onClick={handleAuthorizeWithNonce}
							style={{
								padding: "12px 24px",
								background: "#28a745",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITH nonce (Protected)
						</button>
					</div>
				</div>
			)}

			{phase === "authorize" && (
				<div>
					<h2>Step 2: Token Exchange</h2>
					{result && (
						<div
							style={{
								background: "#f8f9fa",
								padding: 16,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<strong>Server Response:</strong>
							<p>{result.message}</p>
							{result.explanation && (
								<p style={{ color: "#666" }}>{result.explanation}</p>
							)}
						</div>
					)}
					<p>
						Authorization Code: <code>{authCode}</code>
					</p>
					{nonceValue && (
						<p>
							Nonce: <code>{nonceValue}</code>
						</p>
					)}
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={() => handleTokenExchange(false)}
							style={{
								padding: "12px 24px",
								background: "#ffc107",
								color: "black",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Exchange WITHOUT nonce validation
						</button>
						<button
							onClick={() => handleTokenExchange(true)}
							disabled={!nonceValue}
							style={{
								padding: "12px 24px",
								background: nonceValue ? "#28a745" : "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Exchange WITH nonce validation
						</button>
					</div>
				</div>
			)}

			{phase === "token" && tokenResponse && (
				<div>
					<h2>Step 3: ID Token Received</h2>
					<div
						style={{
							background: tokenResponse.defeated ? "#d4edda" : "#f8f9fa",
							border: `1px solid ${tokenResponse.defeated ? "#c3e6cb" : "#dee2e6"}`,
							padding: 16,
							borderRadius: 8,
							marginBottom: 16,
						}}
					>
						<strong>
							{tokenResponse.defeated ? "BOSS DEFEATED!" : "Token Response:"}
						</strong>
						<p>{tokenResponse.message}</p>
						{tokenResponse.explanation && (
							<p style={{ color: "#666", whiteSpace: "pre-line" }}>
								{tokenResponse.explanation}
							</p>
						)}
						{tokenResponse.rfc_link && (
							<p>
								<a
									href={tokenResponse.rfc_link}
									target="_blank"
									rel="noopener noreferrer"
								>
									OpenID Connect Core - Nonce
								</a>
							</p>
						)}
					</div>
					{idToken && (
						<div
							style={{
								background: "#e2e3e5",
								padding: 12,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<p>
								<strong>ID Token:</strong>
							</p>
							<code style={{ wordBreak: "break-all", fontSize: 12 }}>
								{idToken}
							</code>
						</div>
					)}
					{!tokenResponse.defeated && idToken && (
						<button
							onClick={handleReplay}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
								marginRight: 16,
							}}
						>
							Attempt Replay Attack
						</button>
					)}
					<button
						onClick={handleReset}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Try Again
					</button>
				</div>
			)}

			{phase === "replay" && result && (
				<div>
					<h2>Replay Result</h2>
					<div
						style={{
							background: result.success ? "#f8d7da" : "#d4edda",
							border: `1px solid ${result.success ? "#f5c6cb" : "#c3e6cb"}`,
							padding: 16,
							borderRadius: 8,
							marginBottom: 16,
						}}
					>
						<strong>
							{result.success
								? "Replay Attack Succeeded!"
								: "Replay Attack Blocked!"}
						</strong>
						<p>{result.message}</p>
						{result.explanation && (
							<p style={{ color: "#666", whiteSpace: "pre-line" }}>
								{result.explanation}
							</p>
						)}
						{result.rfc_link && (
							<p>
								<a
									href={result.rfc_link}
									target="_blank"
									rel="noopener noreferrer"
								>
									OpenID Connect Core - Nonce
								</a>
							</p>
						)}
					</div>
					<button
						onClick={handleReset}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Try Again
					</button>
				</div>
			)}
		</div>
	);
}
