import { useState } from "react";
import { useBossProgress } from "../../hooks/useBossProgress";
import type { BossResult } from "../../types/boss";

const API_BASE = "/api/boss/2";

type Phase = "intro" | "authorize" | "attack" | "callback" | "result";

export function Boss2Page() {
	const { isDefeated, markDefeated } = useBossProgress();
	const [phase, setPhase] = useState<Phase>("intro");
	const [authCode, setAuthCode] = useState("");
	const [stateParam, setStateParam] = useState("");
	const [result, setResult] = useState<BossResult | null>(null);

	const defeated = isDefeated(2);

	const handleAuthorizeWithoutState = async () => {
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
		if (data.code) {
			setAuthCode(data.code);
		}
		setStateParam("");
		setPhase("authorize");
	};

	const handleAuthorizeWithState = async () => {
		const stateRes = await fetch(`${API_BASE}/generate-state`, {
			method: "POST",
		});
		const stateData: { state: string } = await stateRes.json();
		const generatedState = stateData.state;
		setStateParam(generatedState);

		const res = await fetch(`${API_BASE}/authorize`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				client_id: "demo-client",
				redirect_uri: "http://localhost:3000/callback",
				state: generatedState,
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.code) {
			setAuthCode(data.code);
		}
		setPhase("authorize");
	};

	const handleSimulateAttack = async () => {
		const res = await fetch(`${API_BASE}/attack`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				attacker_code: "attacker-injected-code",
				victim_state: stateParam,
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		setPhase("attack");
	};

	const handleCallback = async (withState: boolean) => {
		const body: Record<string, string> = { code: authCode };
		if (withState && stateParam) {
			body.returned_state = stateParam;
			body.original_state = stateParam;
		}

		const res = await fetch(`${API_BASE}/callback`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(body),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.defeated) {
			markDefeated(2);
		}
		setPhase("result");
	};

	const handleReset = () => {
		setPhase("intro");
		setAuthCode("");
		setStateParam("");
		setResult(null);
	};

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>Boss 2: State Mismatch (CSRF)</h1>
			<p>
				Cross-Site Request Forgery via missing state parameter — RFC 6749
				Section 10.12 requires state for CSRF protection.
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
					<p>Choose how to start the OAuth flow:</p>
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={handleAuthorizeWithoutState}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITHOUT state (Vulnerable)
						</button>
						<button
							onClick={handleAuthorizeWithState}
							style={{
								padding: "12px 24px",
								background: "#28a745",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITH state (Protected)
						</button>
					</div>
				</div>
			)}

			{phase === "authorize" && (
				<div>
					<h2>Step 2: Attacker Injects Code or Proceed to Callback</h2>
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
					{stateParam && (
						<p>
							State Parameter: <code>{stateParam}</code>
						</p>
					)}
					<div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
						<button
							onClick={handleSimulateAttack}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Simulate CSRF Attack
						</button>
						<button
							onClick={() => handleCallback(false)}
							style={{
								padding: "12px 24px",
								background: "#ffc107",
								color: "black",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Callback WITHOUT state validation
						</button>
						<button
							onClick={() => handleCallback(true)}
							disabled={!stateParam}
							style={{
								padding: "12px 24px",
								background: stateParam ? "#28a745" : "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Callback WITH state validation
						</button>
					</div>
				</div>
			)}

			{phase === "attack" && result && (
				<div>
					<h2>Attack Result</h2>
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
								? "CSRF Attack Succeeded!"
								: "CSRF Attack Blocked!"}
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
									RFC 6749 Section 10.12 - CSRF Protection
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

			{phase === "result" && result && (
				<div>
					<h2>Result</h2>
					<div
						style={{
							background: result.defeated ? "#d4edda" : "#f8d7da",
							border: `1px solid ${result.defeated ? "#c3e6cb" : "#f5c6cb"}`,
							padding: 16,
							borderRadius: 8,
							marginBottom: 16,
						}}
					>
						<strong>
							{result.defeated ? "BOSS DEFEATED!" : "Attack Succeeded..."}
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
									RFC 6749 Section 10.12 - CSRF Protection
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
