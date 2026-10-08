#include <math.h>
#include "include/geometry.h"

double point_distance(Point2D a, Point2D b) {
    return hypot(b.x - a.x, b.y - a.y);
}

double polygon_perimeter(const Point2D *pts, int n) {
    if (n < 2) {
        return 0.0;
    }
    double total = 0.0;
    for (int i = 0; i < n; i++) {
        Point2D next = pts[(i + 1) % n];
        total += point_distance(pts[i], next);
    }
    return total;
}
