import re
from dataclasses import dataclass

from agent_engine.errors import InvalidPlanError
from agent_engine.models import RelevantFile
from agent_engine.repository import normalize_tool_path

TITLE = "# Implementation Plan"
SECTIONS = [
    "Issue Summary",
    "Goal",
    "Relevant Files",
    "Proposed Changes",
    "Implementation Steps",
    "Expected Files to Change",
    "Tests and Verification",
    "Risks and Open Questions",
]
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*$", re.MULTILINE)
FILE_BULLET_RE = re.compile(
    r"^\s*[-*]\s+`(?P<path>[^`]+)`(?P<create>\s*\(create\))?\s*:\s*(?P<reason>.+?)\s*$",
    re.MULTILINE,
)


@dataclass(frozen=True)
class ValidatedPlan:
    markdown: str
    relevant_files: list[RelevantFile]


class PlanValidator:
    def validate(self, markdown: str, available_paths: set[str]) -> ValidatedPlan:
        candidate = markdown.strip()
        errors = self.collect_errors(candidate, available_paths)
        if errors:
            raise InvalidPlanError("; ".join(errors))
        sections = self._split_sections(candidate)
        relevant_files = self._parse_file_bullets(sections["Relevant Files"])
        return ValidatedPlan(markdown=candidate, relevant_files=relevant_files)

    def collect_errors(self, markdown: str, available_paths: set[str]) -> list[str]:
        errors = []
        if not markdown:
            return ["output is empty"]
        if len(markdown) > 200_000:
            errors.append("plan exceeds 200000 characters")
        if "```" in markdown:
            errors.append("fenced code blocks are not allowed")
        if not markdown.startswith(f"{TITLE}\n"):
            errors.append(f"plan must start with {TITLE}")

        headings = [(len(match[0]), match[1]) for match in HEADING_RE.findall(markdown)]
        expected = [(1, "Implementation Plan"), *((2, section) for section in SECTIONS)]
        if headings != expected:
            errors.append("headings are missing, duplicated, or out of order")
            return errors

        sections = self._split_sections(markdown)
        for name in SECTIONS:
            if not sections.get(name, "").strip():
                errors.append(f"section {name} is empty")

        if sections.get("Implementation Steps") and not re.search(
            r"^\s*1\.\s+\S", sections["Implementation Steps"], re.MULTILINE
        ):
            errors.append("Implementation Steps must contain an ordered list")

        for section_name in ("Relevant Files", "Expected Files to Change"):
            body = sections.get(section_name, "")
            bullets = self._parse_file_bullets(body)
            if not bullets:
                errors.append(f"section {section_name} must contain path bullets")
                continue
            for item in bullets:
                try:
                    normalize_tool_path(item.path)
                except ValueError:
                    errors.append(f"plan contains unsafe repository path: {item.path}")
                    continue
                if item.create:
                    continue
                if item.path not in available_paths:
                    errors.append(f"plan references unavailable repository file: {item.path}")
        return errors

    def _split_sections(self, markdown: str) -> dict[str, str]:
        positions = list(HEADING_RE.finditer(markdown))
        result = {}
        for index, match in enumerate(positions):
            name = match.group(2)
            start = match.end()
            end = positions[index + 1].start() if index + 1 < len(positions) else len(markdown)
            result[name] = markdown[start:end].strip()
        return result

    def _parse_file_bullets(self, body: str) -> list[RelevantFile]:
        return [
            RelevantFile(
                path=match.group("path"),
                reason=match.group("reason"),
                create=bool(match.group("create")),
            )
            for match in FILE_BULLET_RE.finditer(body)
        ]
