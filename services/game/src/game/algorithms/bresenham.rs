pub struct Bresenham {
    row: u16,
    col: u16,
    row1: u16,
    col1: u16,
    step_row: i32,
    step_col: i32,
    drow: i32,
    dcol: i32,
    error: i32,
    steep: bool,
    finished: bool,
}

impl Bresenham {
    pub fn new(p0: (u16, u16), p1: (u16, u16)) -> Bresenham {
        let drow = p1.0 as i32 - p0.0 as i32;
        let dcol = p1.1 as i32 - p0.1 as i32;
        let steep = drow.abs() > dcol.abs();

        let drow_abs = drow.abs();
        let dcol_abs = dcol.abs();

        let error = if steep {
            2 * dcol_abs - drow_abs
        } else {
            2 * drow_abs - dcol_abs
        };

        Bresenham {
            row: p0.0,
            col: p0.1,
            row1: p1.0,
            col1: p1.1,
            step_row: if p0.0 < p1.0 { 1 } else { -1 },
            step_col: if p0.1 < p1.1 { 1 } else { -1 },
            drow: drow_abs,
            dcol: dcol_abs,
            error,
            steep,
            finished: false,
        }
    }
}

impl Iterator for Bresenham {
    type Item = (u16, u16);

    fn next(&mut self) -> Option<Self::Item> {
        if self.finished {
            return None;
        }

        let current_pos = (self.row, self.col);

        if current_pos == (self.row1, self.col1) {
            self.finished = true;
            return Some(current_pos);
        }

        if self.steep {
            if self.error >= 0 {
                self.col = (self.col as i32 + self.step_col) as u16;
                self.error -= 2 * self.drow;
            }
            self.row = (self.row as i32 + self.step_row) as u16;
            self.error += 2 * self.dcol;
        } else {
            if self.error >= 0 {
                self.row = (self.row as i32 + self.step_row) as u16;
                self.error -= 2 * self.dcol;
            }
            self.col = (self.col as i32 + self.step_col) as u16;
            self.error += 2 * self.drow;
        }
        Some(current_pos)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_initial_state() {
        let start = (0, 0);
        let end = (2, 5);
        let bresenham = Bresenham::new(start, end);

        assert_eq!(bresenham.row, 0, "Initial row should be the start row");
        assert_eq!(bresenham.col, 0, "Initial col should be the start col");
        assert_eq!(bresenham.row1, 2, "row1 should be the end row");
        assert_eq!(bresenham.col1, 5, "col1 should be the end col");
    }

    #[test]
    fn test_horizontal_line() {
        let start = (0, 0);
        let end = (0, 5);
        let bresenham = Bresenham::new(start, end);
        let path: Vec<_> = bresenham.collect();
        assert_eq!(path, vec![(0, 0), (0, 1), (0, 2), (0, 3), (0, 4), (0, 5)]);
    }

    #[test]
    fn test_vertical_line() {
        let start = (0, 0);
        let end = (5, 0);
        let bresenham = Bresenham::new(start, end);
        let path: Vec<_> = bresenham.collect();
        assert_eq!(path, vec![(0, 0), (1, 0), (2, 0), (3, 0), (4, 0), (5, 0)]);
    }

    #[test]
    fn test_diagonal_line_shallow() {
        let start = (0, 0);
        let end = (2, 5);
        let bresenham = Bresenham::new(start, end);
        let path: Vec<_> = bresenham.collect();
        assert_eq!(path, vec![(0, 0), (0, 1), (1, 2), (1, 3), (2, 4), (2, 5)]);
    }

    #[test]
    fn test_diagonal_line_steep() {
        let start = (0, 0);
        let end = (5, 2);
        let bresenham = Bresenham::new(start, end);
        let path: Vec<_> = bresenham.collect();
        assert_eq!(path, vec![(0, 0), (1, 0), (2, 1), (3, 1), (4, 2), (5, 2)]);
    }

    #[test]
    fn test_diagonal_line_going_left() {
        let start = (0, 5);
        let end = (2, 0);
        let bresenham = Bresenham::new(start, end);
        let path: Vec<_> = bresenham.collect();
        assert_eq!(path, vec![(0, 5), (0, 4), (1, 3), (1, 2), (2, 1), (2, 0)]);
    }
}
