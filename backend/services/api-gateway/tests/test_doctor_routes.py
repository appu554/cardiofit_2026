import pytest
import httpx
from app.main import app


@pytest.mark.anyio
async def test_doctor_summary_requires_auth():
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/v1/doctor/patients/p1/summary")
        assert resp.status_code == 401


@pytest.mark.anyio
async def test_doctor_graphql_requires_auth():
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/v1/doctor/graphql", json={"query": "{ __typename }"})
        assert resp.status_code == 401
