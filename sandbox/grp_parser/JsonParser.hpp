//
// Created by User on 2025/7/26.
//

#ifndef JSONPARSER_HPP
#define JSONPARSER_HPP

#include <fstream>
#include <nlohmann/json.hpp>
#include <string>
#include <vector>
#include <iostream>
#include <unordered_map>

using namespace std;
using JSON = nlohmann::json;

class JSONParser{
private:
    string parse_path;              // All JSON file path
    string score_path;              // Score file path
    JSON scoref;                    // SCORE JSON file path
    JSON utf;                       // JSON file path
    double score = 0.0;             // Score
    unordered_map<string,int> task; // Score for every testsuite


    void ParseScore();
public:
    JSONParser(string parse_path,string score_path);
    void Parse();
    double Getscore();
};

#endif
