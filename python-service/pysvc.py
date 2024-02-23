from typing import Annotated
import fastapi
from fastapi import Depends
from elasticapm.contrib.starlette import make_apm_client, ElasticAPM
import httpx

apm = make_apm_client()
app = fastapi.FastAPI(
    title="pysvc",
    description="just a pointless http python service",
)
app.add_middleware(ElasticAPM, client=apm)


@app.get("/healthz")
async def healthz():
    return {"jut?": "juuuut!"}


async def get_client():
    async with httpx.AsyncClient() as client:
        yield client

@app.get("/dice")
async def proxy_dice(client: Annotated[httpx.AsyncClient, Depends(get_client)]):
        resp = await client.get("http://dice:8080/rolldice")
        return await resp.aread()
