#include "JsonParser.hpp"

JSONParser::JSONParser(std::string parse_path, std::string score_path)
        : parse_path(parse_path), score_path(score_path) {
    ifstream i(parse_path);
    utf << i;
    i.close();
    i.open(score_path);
    scoref << i;
    ParseScore();
}
void JSONParser::ParseScore() {
    auto score_array = scoref["testsuites"];
    for(auto i:score_array){
        string name = i["testsuite"];
        double sc = i["score"];
        task[name] = sc * 1.0;
    }
}

void JSONParser::Parse() {
    auto testSuites = utf["testsuites"];
    for(auto test:testSuites){
        string name = test["name"];
        double ac = 0.0,wa = 0.0;
        auto suites = test["testsuite"];
        for(auto suite:suites){
            if(suite.contains("failures") || suite.contains("errors")){
                wa += 1.0;
            } else {
                ac += 1.0;
            }
        }
        this->score += ac/(ac+wa) * task[name];
    }
}

double JSONParser::Getscore() {
    return score;
}