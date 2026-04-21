import httpx
from typing import Optional

class HTTPClient:
    def __init__(self, timeout: int = 30, max_retries: int = 3):
        self.timeout = timeout
        self.max_retries = max_retries
        self.client = httpx.AsyncClient(timeout=timeout)

    async def fetch(self, url: str) -> Optional[bytes]:
        """Faz download de URL"""
        for attempt in range(self.max_retries):
            try:
                response = await self.client.get(url)
                response.raise_for_status()
                return response.content
            except httpx.HTTPError as e:
                if attempt == self.max_retries - 1:
                    raise
                continue
        return None

    async def close(self):
        await self.client.aclose()
