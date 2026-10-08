#ifndef GEOMETRY_H
#define GEOMETRY_H

typedef struct {
    double x;
    double y;
} Point2D;

double point_distance(Point2D a, Point2D b);
double polygon_perimeter(const Point2D *pts, int n);

#endif
