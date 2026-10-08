#include <math.h>
#include "include/stats.h"

double array_mean(const double *values, int n) {
    if (n <= 0) {
        return 0.0;
    }
    double sum = 0.0;
    for (int i = 0; i < n; i++) {
        sum += values[i];
    }
    return sum / (double)n;
}

double array_stddev(const double *values, int n) {
    if (n <= 0) {
        return 0.0;
    }
    double mean = array_mean(values, n);
    double sq_diff_sum = 0.0;
    for (int i = 0; i < n; i++) {
        double diff = values[i] - mean;
        sq_diff_sum += diff * diff;
    }
    return sqrt(sq_diff_sum / (double)n);
}
