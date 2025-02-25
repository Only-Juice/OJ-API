import json
import sys
from enum import Enum

from art import text2art


class ScoreMethod(Enum):
    SUITE = 1
    TEST = 2


class grp_parser:
    def __init__(self, path, scoremethod=ScoreMethod.TEST):
        self.path = path
        self.data = self.load()
        self.__scoremethod: ScoreMethod = scoremethod
        self.__calculate_score()

    def __calculate_score(self):
        testsuites = self.data["testsuites"]
        self.__total_tests_count = 0
        self.__passed_tests = []
        self.__failures_tests = []
        self.__failures_suites = set()
        for testsuite in testsuites:
            self.__total_tests_count += len(testsuite["testsuite"])
            for testcase in testsuite["testsuite"]:
                if "failures" not in testcase:
                    self.__passed_tests.append(testcase["name"])
                else:
                    self.__failures_tests.append(testcase["name"])
                    self.__failures_suites.add(testsuite["name"])
        if self.__scoremethod == ScoreMethod.SUITE:
            self.__score = 100 * (1 - len(self.__failures_suites) / len(testsuites))
        elif self.__scoremethod == ScoreMethod.TEST:
            self.__score = 100 * (len(self.__passed_tests) / self.__total_tests_count)

    def load(self):
        with open(self.path, "r") as f:
            return json.load(f)

    def parser(self, color=False):
        def colorize(statusOK=True, text=""):
            if color:
                return f"\033[{32 if statusOK else 31}m{text}\033[0m"
            return text

        testsuites = self.data["testsuites"]
        result = (
            text2art("NTUT-OOPOJ", font="lildevil") + "\n"
            f"{colorize(text='[==========]')} Running {self.data['tests']} tests from "
            f"{len(testsuites)} test suites.\n"
            f"{colorize(text='[----------]')} Global test environment set-up.\n"
        )
        for testsuite in testsuites:
            result += (
                f"{colorize(text='[----------]')} {len(testsuite['testsuite'])} tests from"
                f" {testsuite['name']}"
            )
            if self.__scoremethod == ScoreMethod.SUITE:
                result += f"({100 * (1 / len(testsuites)):.1f}pt)"
            elif self.__scoremethod == ScoreMethod.TEST:
                result += f"({100 * (len(testsuite['testsuite']) / self.__total_tests_count):.1f}pt)"
            result += "\n"
            for testcase in testsuite["testsuite"]:
                status = f"[ {testcase['status']:8} ]"
                result += f"{colorize(text=status)} {testcase['name']}"
                if self.__scoremethod == ScoreMethod.TEST:
                    result += f"({100 * (1 / self.__total_tests_count):.1f}pt)"
                result += "\n"
                if "failures" not in testcase:
                    result += f"{colorize(text='[       OK ]')} {testcase['name']} ({testcase['time']})\n"
                else:
                    for failure in testcase["failures"]:
                        result += failure["failure"] + "\n"
                    result += f"{colorize(statusOK=False, text='[  FAILED  ]')} {testcase['name']}\n"
            result += (
                f"{colorize(text='[----------]')} {len(testsuite['testsuite'])} tests from"
                f" {testsuite['name']} (0 ms total)\n\n"
            )

        result += (
            f"{colorize(text='[----------]')} Global test environment tear-down\n"
            f"{colorize(text='[==========]')} Running {self.__total_tests_count} tests from"
            f" {len(testsuites)} test suites.\n"
        )
        if self.__passed_tests:
            result += (
                f"{colorize(statusOK=True, text='[  PASSED  ]')} {len(self.__passed_tests)} "
                f"tests.\n"
            )
        if self.__failures_tests:
            result += (
                f"{colorize(statusOK=False, text='[  FAILED  ]')} {len(self.__failures_tests)} "
                f"tests, listed below:\n"
            )
            for failed_test in self.__failures_tests:
                result += (
                    f"{colorize(statusOK=False, text='[  FAILED  ]')} {failed_test}\n"
                )
            result += f"\n{len(self.__failures_tests)} FAILED TESTS\n"
        result += text2art(f"Score: {self.__score:.1f}", font="doom")
        return result

    def get_score(self):
        return self.__score


def __main():
    if len(sys.argv) != 2:
        print("Usage: python grp-parser.py <path to json>")

    parser = grp_parser(sys.argv[1])
    print(parser.parser(color=True))
    # save score to score.txt
    with open("score.txt", "w") as f:
        f.write(f"{parser.get_score():.1f}")
    print("Score has been saved to score.txt")
    # save output to message.txt
    with open("message.txt", "w") as f:
        f.write(parser.parser())
    print("Output has been saved to message.txt")


if __name__ == "__main__":
    __main()
